// Package store persists sur session/task state in SQLite.
// Uses modernc.org/sqlite (pure-Go, CGO-free) so binaries stay static.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DefaultPath is where sur stores its database on a real system.
const DefaultPath = "/var/lib/sur/sur.db"

// Store wraps the SQLite database.
type Store struct {
	db   *sql.DB
	path string
}

// Open opens (and creates if needed) the SQLite database at path.
// Pass an empty string to use DefaultPath, prefixed with $SUR_DB if set.
func Open(path string) (*Store, error) {
	if path == "" {
		if env := os.Getenv("SUR_DB"); env != "" {
			path = env
		} else {
			path = DefaultPath
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil { // #nosec G703 -- path is the hardcoded DefaultPath or operator-set SUR_DB env var
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	s := &Store{db: db, path: path}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

// Path returns the file location used by this store.
func (s *Store) Path() string { return s.path }

// Close releases the underlying database handle.
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS apply_sessions (
    id          TEXT PRIMARY KEY,
    hostname    TEXT,
    started_at  DATETIME,
    finished_at DATETIME,
    status      TEXT
);

CREATE TABLE IF NOT EXISTS task_executions (
    id                TEXT PRIMARY KEY,
    session_id        TEXT REFERENCES apply_sessions(id) ON DELETE CASCADE,
    task_id           TEXT,
    status            TEXT,
    backup_data       BLOB,
    backup_path       TEXT,
    rollback_possible INTEGER,
    executed_at       DATETIME,
    error_message     TEXT
);

CREATE INDEX IF NOT EXISTS idx_task_session ON task_executions(session_id);
`
	_, err := s.db.Exec(schema)
	return err
}

// ---------- domain types ----------

// SessionStatus enumerates apply_sessions.status values.
type SessionStatus string

const (
	SessionRunning   SessionStatus = "running"
	SessionCompleted SessionStatus = "completed"
	SessionFailed    SessionStatus = "failed"
	SessionPartial   SessionStatus = "partial"
)

// TaskStatus enumerates task_executions.status values.
type TaskStatus string

const (
	TaskSuccess    TaskStatus = "success"
	TaskFailed     TaskStatus = "failed"
	TaskRolledBack TaskStatus = "rolled_back"
	TaskSkipped    TaskStatus = "skipped"
)

// Session row.
type Session struct {
	ID         string
	Hostname   string
	StartedAt  time.Time
	FinishedAt sql.NullTime
	Status     SessionStatus
}

// TaskExecution row.
type TaskExecution struct {
	ID               string
	SessionID        string
	TaskID           string
	Status           TaskStatus
	BackupData       []byte
	BackupPath       string
	RollbackPossible bool
	ExecutedAt       time.Time
	ErrorMessage     string
}

// ---------- CRUD ----------

// CreateSession inserts a new apply_sessions row in "running" state.
func (s *Store) CreateSession(sess Session) error {
	_, err := s.db.Exec(
		`INSERT INTO apply_sessions(id, hostname, started_at, status) VALUES(?,?,?,?)`,
		sess.ID, sess.Hostname, sess.StartedAt.UTC(), string(SessionRunning),
	)
	return err
}

// FinishSession updates status and finished_at.
func (s *Store) FinishSession(id string, status SessionStatus) error {
	_, err := s.db.Exec(
		`UPDATE apply_sessions SET status=?, finished_at=? WHERE id=?`,
		string(status), time.Now().UTC(), id,
	)
	return err
}

// GetSession fetches one session by id.
func (s *Store) GetSession(id string) (*Session, error) {
	row := s.db.QueryRow(`SELECT id, hostname, started_at, finished_at, status FROM apply_sessions WHERE id=?`, id)
	var sess Session
	var status string
	if err := row.Scan(&sess.ID, &sess.Hostname, &sess.StartedAt, &sess.FinishedAt, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	sess.Status = SessionStatus(status)
	return &sess, nil
}

// ListSessions returns sessions, newest first.
func (s *Store) ListSessions(limit int) ([]Session, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(
		`SELECT id, hostname, started_at, finished_at, status FROM apply_sessions ORDER BY started_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		var sess Session
		var status string
		if err := rows.Scan(&sess.ID, &sess.Hostname, &sess.StartedAt, &sess.FinishedAt, &status); err != nil {
			return nil, err
		}
		sess.Status = SessionStatus(status)
		out = append(out, sess)
	}
	return out, rows.Err()
}

// RecordTask stores a task execution row.
func (s *Store) RecordTask(t TaskExecution) error {
	_, err := s.db.Exec(
		`INSERT INTO task_executions(
            id, session_id, task_id, status, backup_data, backup_path,
            rollback_possible, executed_at, error_message
        ) VALUES(?,?,?,?,?,?,?,?,?)`,
		t.ID, t.SessionID, t.TaskID, string(t.Status), t.BackupData, t.BackupPath,
		boolToInt(t.RollbackPossible), t.ExecutedAt.UTC(), t.ErrorMessage,
	)
	return err
}

// UpdateTaskStatus mutates an existing task row.
func (s *Store) UpdateTaskStatus(id string, status TaskStatus, errMsg string) error {
	_, err := s.db.Exec(
		`UPDATE task_executions SET status=?, error_message=? WHERE id=?`,
		string(status), errMsg, id,
	)
	return err
}

// TasksForSession returns task executions ordered by executed_at ASC.
func (s *Store) TasksForSession(sessionID string) ([]TaskExecution, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, task_id, status, backup_data, backup_path,
                rollback_possible, executed_at, error_message
         FROM task_executions WHERE session_id=? ORDER BY executed_at ASC`,
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TaskExecution
	for rows.Next() {
		var t TaskExecution
		var status string
		var rp int
		if err := rows.Scan(
			&t.ID, &t.SessionID, &t.TaskID, &status, &t.BackupData, &t.BackupPath,
			&rp, &t.ExecutedAt, &t.ErrorMessage,
		); err != nil {
			return nil, err
		}
		t.Status = TaskStatus(status)
		t.RollbackPossible = rp == 1
		out = append(out, t)
	}
	return out, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
