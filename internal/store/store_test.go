package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "sur.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestSessionLifecycle(t *testing.T) {
	s := newStore(t)
	id := uuid.NewString()
	if err := s.CreateSession(Session{ID: id, Hostname: "test", StartedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetSession(id)
	if err != nil || got == nil {
		t.Fatalf("GetSession err=%v got=%v", err, got)
	}
	if got.Status != SessionRunning {
		t.Fatalf("status = %s", got.Status)
	}
	if err := s.FinishSession(id, SessionCompleted); err != nil {
		t.Fatal(err)
	}
	got, _ = s.GetSession(id)
	if got.Status != SessionCompleted {
		t.Fatalf("after finish status = %s", got.Status)
	}
}

func TestTaskRecording(t *testing.T) {
	s := newStore(t)
	sid := uuid.NewString()
	_ = s.CreateSession(Session{ID: sid, Hostname: "h", StartedAt: time.Now()})

	tid := uuid.NewString()
	err := s.RecordTask(TaskExecution{
		ID: tid, SessionID: sid, TaskID: "disable_root_ssh",
		Status: TaskSuccess, RollbackPossible: true,
		BackupPath: "/etc/ssh/sshd_config", BackupData: []byte("orig"),
		ExecutedAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	tasks, err := s.TasksForSession(sid)
	if err != nil || len(tasks) != 1 {
		t.Fatalf("tasks = %d err = %v", len(tasks), err)
	}
	if !tasks[0].RollbackPossible || string(tasks[0].BackupData) != "orig" {
		t.Fatalf("unexpected task: %+v", tasks[0])
	}
}

func TestListSessionsNewestFirst(t *testing.T) {
	s := newStore(t)
	for i := 0; i < 3; i++ {
		_ = s.CreateSession(Session{
			ID: uuid.NewString(), Hostname: "h",
			StartedAt: time.Now().Add(time.Duration(i) * time.Second),
		})
	}
	got, err := s.ListSessions(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3, got %d", len(got))
	}
	if !got[0].StartedAt.After(got[1].StartedAt) {
		t.Fatal("not ordered newest first")
	}
}
