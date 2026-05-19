// Package engine loads YAML task definitions and runs the
// pre_check → backup → exec → post_check → rollback lifecycle.
package engine

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/suleymanmercan/sur/internal/store"
)

// Step is a single shell command inside a task phase.
type Step struct {
	Command    string `yaml:"command"`
	ExpectExit *int   `yaml:"expect_exit,omitempty"`
}

// Phase is a list of steps.
type Phase []Step

// PreCheck describes a single command + expected exit code.
type PreCheck struct {
	Command    string `yaml:"command"`
	ExpectExit int    `yaml:"expect_exit"`
}

// Task is the YAML schema for a hardening task.
type Task struct {
	ID               string   `yaml:"id"`
	Name             string   `yaml:"name"`
	Description      string   `yaml:"description"`
	RollbackPossible bool     `yaml:"rollback_possible"`
	BackupFiles      []string `yaml:"backup_files"`
	RiskLevel        string   `yaml:"risk_level"`
	Distros          []string `yaml:"distros"`
	PreCheck         PreCheck `yaml:"pre_check"`
	Exec             Phase    `yaml:"exec"`
	PostCheck        PreCheck `yaml:"post_check"`
	Rollback         Phase    `yaml:"rollback"`

	// SourcePath is populated by LoadTasks; not part of YAML schema.
	SourcePath string `yaml:"-"`
}

// LoadTasks parses every *.yaml file under dir and returns them sorted by ID.
func LoadTasks(dir string) ([]Task, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	yml, _ := filepath.Glob(filepath.Join(dir, "*.yml"))
	matches = append(matches, yml...)

	var tasks []Task
	for _, m := range matches {
		t, err := loadTaskFile(m)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", m, err)
		}
		tasks = append(tasks, t)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	return tasks, nil
}

// LoadTasksFS reads tasks from an embedded filesystem.
func LoadTasksFS(fs embed.FS, dir string) ([]Task, error) {
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var tasks []Task
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		b, err := fs.ReadFile(dir + "/" + name)
		if err != nil {
			return nil, err
		}
		var t Task
		if err := yaml.Unmarshal(b, &t); err != nil {
			return nil, err
		}
		if t.ID == "" {
			return nil, fmt.Errorf("%s: task id is required", name)
		}
		tasks = append(tasks, t)
	}
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })
	return tasks, nil
}
func loadTaskFile(path string) (Task, error) {
	var t Task
	b, err := os.ReadFile(path)
	if err != nil {
		return t, err
	}
	if err := yaml.Unmarshal(b, &t); err != nil {
		return t, err
	}
	if t.ID == "" {
		return t, fmt.Errorf("task id is required")
	}
	t.SourcePath = path
	return t, nil
}

// ---------- runtime ----------

// Result describes what happened to a single task.
type Result struct {
	TaskID   string
	Status   store.TaskStatus
	Err      error
	Duration time.Duration
}

// NeedsRun reports whether a task's pre_check says there is work to do.
func NeedsRun(ctx context.Context, t Task) (bool, int) {
	if t.PreCheck.Command == "" {
		return true, 0
	}
	_, code := runShell(ctx, t.PreCheck.Command)
	return code == t.PreCheck.ExpectExit, code
}

// Runner executes tasks and records state.
type Runner struct {
	Store    *store.Store
	DryRun   bool
	Hostname string
	// Logger receives human-readable progress messages.
	Logger func(format string, args ...any)
}

func (r *Runner) log(format string, args ...any) {
	if r.Logger != nil {
		r.Logger(format, args...)
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// StartSession opens a new persistent session and returns its ID.
func (r *Runner) StartSession() (string, error) {
	id := uuid.NewString()
	host, _ := os.Hostname()
	if r.Hostname != "" {
		host = r.Hostname
	}
	return id, r.Store.CreateSession(store.Session{ID: id, Hostname: host, StartedAt: time.Now()})
}

// Apply runs the given tasks under sessionID. It never aborts the whole run:
// per-task failures rollback that task only.
func (r *Runner) Apply(ctx context.Context, sessionID string, tasks []Task) []Result {
	var results []Result
	anyFailed := false
	anyDid := false
	for _, t := range tasks {
		select {
		case <-ctx.Done():
			results = append(results, Result{TaskID: t.ID, Status: store.TaskSkipped, Err: ctx.Err()})
			continue
		default:
		}
		res := r.runTask(ctx, sessionID, t)
		results = append(results, res)
		switch res.Status {
		case store.TaskFailed, store.TaskRolledBack:
			anyFailed = true
			anyDid = true
		case store.TaskSuccess:
			anyDid = true
		}
	}

	status := store.SessionCompleted
	switch {
	case anyFailed && anyDid:
		status = store.SessionPartial
	case anyFailed:
		status = store.SessionFailed
	}
	_ = r.Store.FinishSession(sessionID, status)
	return results
}

func (r *Runner) runTask(ctx context.Context, sessionID string, t Task) Result {
	start := time.Now()
	execID := uuid.NewString()

	rec := store.TaskExecution{
		ID: execID, SessionID: sessionID, TaskID: t.ID,
		RollbackPossible: t.RollbackPossible, ExecutedAt: time.Now(),
		Status: store.TaskSuccess,
	}

	// 1. pre_check (informational; skip when already satisfied)
	needsRun, code := NeedsRun(ctx, t)
	if !needsRun {
		msg := fmt.Sprintf("already satisfied or not applicable (pre_check exit %d, expected %d)", code, t.PreCheck.ExpectExit)
		if t.PreCheck.Command != "" {
			r.log("[%s] %s", t.ID, msg)
			rec.Status = store.TaskSkipped
			rec.ErrorMessage = msg
			_ = r.Store.RecordTask(rec)
			return Result{TaskID: t.ID, Status: store.TaskSkipped, Err: fmt.Errorf("%s", msg), Duration: time.Since(start)}
		}
	}

	if r.DryRun {
		r.log("[%s] DRY-RUN — would run %d step(s)", t.ID, len(t.Exec))
		for _, s := range t.Exec {
			r.log("       $ %s", s.Command)
		}
		rec.Status = store.TaskSkipped
		rec.ErrorMessage = "dry-run"
		_ = r.Store.RecordTask(rec)
		return Result{TaskID: t.ID, Status: store.TaskSkipped, Duration: time.Since(start)}
	}

	// 2. backup files
	var backupPath string
	var backupBlob []byte
	for _, f := range t.BackupFiles {
		if data, err := os.ReadFile(f); err == nil {
			backupBlob = data
			backupPath = f
			break
		}
	}
	rec.BackupData = backupBlob
	rec.BackupPath = backupPath

	// 3. exec
	r.log("[%s] applying...", t.ID)
	var execErr error
	for _, s := range t.Exec {
		out, code := runShell(ctx, s.Command)
		expected := 0
		if s.ExpectExit != nil {
			expected = *s.ExpectExit
		}
		if code != expected {
			execErr = fmt.Errorf("step %q exited %d: %s", s.Command, code, strings.TrimSpace(out))
			break
		}
	}

	// 4. post_check
	if execErr == nil && t.PostCheck.Command != "" {
		_, code := runShell(ctx, t.PostCheck.Command)
		if code != t.PostCheck.ExpectExit {
			execErr = fmt.Errorf("post_check failed (exit %d)", code)
		}
	}

	if execErr != nil {
		rec.Status = store.TaskFailed
		rec.ErrorMessage = execErr.Error()
		r.log("[%s] FAILED: %v", t.ID, execErr)
		_ = r.Store.RecordTask(rec)
		// 5. rollback
		if t.RollbackPossible {
			if err := r.rollbackTask(ctx, t, backupPath, backupBlob); err != nil {
				r.log("[%s] rollback error: %v", t.ID, err)
			} else {
				r.log("[%s] rolled back", t.ID)
				_ = r.Store.UpdateTaskStatus(execID, store.TaskRolledBack, execErr.Error())
				return Result{TaskID: t.ID, Status: store.TaskRolledBack, Err: execErr, Duration: time.Since(start)}
			}
		}
		return Result{TaskID: t.ID, Status: store.TaskFailed, Err: execErr, Duration: time.Since(start)}
	}

	r.log("[%s] OK", t.ID)
	_ = r.Store.RecordTask(rec)
	return Result{TaskID: t.ID, Status: store.TaskSuccess, Duration: time.Since(start)}
}

func (r *Runner) rollbackTask(ctx context.Context, t Task, backupPath string, blob []byte) error {
	// Restore the backup file first so rollback commands can rely on it.
	if backupPath != "" && len(blob) > 0 {
		if err := os.WriteFile(backupPath, blob, 0o644); err != nil {
			return fmt.Errorf("restore %s: %w", backupPath, err)
		}
	}
	for _, s := range t.Rollback {
		cmd := strings.ReplaceAll(s.Command, "{backup_path}", backupPath)
		out, code := runShell(ctx, cmd)
		if code != 0 {
			return fmt.Errorf("rollback step %q exit %d: %s", cmd, code, strings.TrimSpace(out))
		}
	}
	return nil
}

// RollbackSession replays the rollback for every task in the session,
// in reverse order. Tasks marked rollback_possible=false are skipped.
//
// Deprecated: RollbackSession can only load tasks from disk and will fail
// on production installs that use the embedded task FS. Use
// RollbackSessionTasks with tasks loaded via engine.LoadTasksFS instead.
func (r *Runner) RollbackSession(ctx context.Context, sessionID string, taskDir string) ([]Result, error) {
	tasks, err := LoadTasks(taskDir)
	if err != nil {
		return nil, err
	}
	return r.RollbackSessionTasks(ctx, sessionID, tasks)
}

// RollbackSessionTasks replays rollback using already loaded task definitions.
func (r *Runner) RollbackSessionTasks(ctx context.Context, sessionID string, tasks []Task) ([]Result, error) {
	byID := map[string]Task{}
	for _, t := range tasks {
		byID[t.ID] = t
	}

	execs, err := r.Store.TasksForSession(sessionID)
	if err != nil {
		return nil, err
	}
	var out []Result
	for i := len(execs) - 1; i >= 0; i-- {
		e := execs[i]
		t, ok := byID[e.TaskID]
		if !ok {
			out = append(out, Result{TaskID: e.TaskID, Status: store.TaskSkipped, Err: fmt.Errorf("task definition not found")})
			continue
		}
		if !e.RollbackPossible {
			out = append(out, Result{TaskID: e.TaskID, Status: store.TaskSkipped, Err: fmt.Errorf("not rollback-able")})
			continue
		}
		if e.Status == store.TaskSkipped || e.Status == store.TaskRolledBack {
			out = append(out, Result{TaskID: e.TaskID, Status: store.TaskSkipped, Err: fmt.Errorf("task status is %s", e.Status)})
			continue
		}
		if err := r.rollbackTask(ctx, t, e.BackupPath, e.BackupData); err != nil {
			out = append(out, Result{TaskID: e.TaskID, Status: store.TaskFailed, Err: err})
			continue
		}
		_ = r.Store.UpdateTaskStatus(e.ID, store.TaskRolledBack, "manual rollback")
		out = append(out, Result{TaskID: e.TaskID, Status: store.TaskRolledBack})
	}
	return out, nil
}

func runShell(ctx context.Context, command string) (string, int) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(out), exitErr.ExitCode()
		}
		return string(out), -1
	}
	return string(out), 0
}
