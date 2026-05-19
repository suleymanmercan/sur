package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/suleymanmercan/sur/internal/store"
)

func writeFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newRunner(t *testing.T) (*Runner, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "sur.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return &Runner{Store: s, Logger: func(string, ...any) {}}, dir
}

func TestLoadTasks(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.yaml"), `
id: task_a
name: A
rollback_possible: true
exec:
  - command: "true"
`)
	writeFile(t, filepath.Join(dir, "b.yaml"), `
id: task_b
name: B
exec:
  - command: "true"
`)
	tasks, err := LoadTasks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 2 || tasks[0].ID != "task_a" {
		t.Fatalf("tasks = %+v", tasks)
	}
}

func TestApply_Success(t *testing.T) {
	r, _ := newRunner(t)
	target := filepath.Join(t.TempDir(), "out")

	sid, err := r.StartSession()
	if err != nil {
		t.Fatal(err)
	}
	results := r.Apply(context.Background(), sid, []RunnableTask{Task{
		ID: "touch_task", Name: "touch",
		Exec:      Phase{{Command: "touch " + target}},
		PostCheck: PreCheck{Command: "test -f " + target, ExpectExit: 0},
	}})
	if len(results) != 1 || results[0].Status != store.TaskSuccess {
		t.Fatalf("results = %+v", results)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}

func TestApply_FailureRollback(t *testing.T) {
	r, _ := newRunner(t)
	backupFile := filepath.Join(t.TempDir(), "cfg")
	writeFile(t, backupFile, "ORIGINAL")

	sid, _ := r.StartSession()
	task := Task{
		ID: "bad_task", Name: "bad", RollbackPossible: true,
		BackupFiles: []string{backupFile},
		Exec: Phase{
			{Command: "echo 'CHANGED' > " + backupFile},
			{Command: "false"}, // force failure
		},
	}
	res := r.Apply(context.Background(), sid, []RunnableTask{task})
	if res[0].Status != store.TaskRolledBack {
		t.Fatalf("expected rolled_back, got %s", res[0].Status)
	}
	got, _ := os.ReadFile(backupFile)
	if string(got) != "ORIGINAL" {
		t.Fatalf("file not restored, content = %q", got)
	}
}

func TestApply_DryRun(t *testing.T) {
	r, _ := newRunner(t)
	r.DryRun = true
	sid, _ := r.StartSession()
	target := filepath.Join(t.TempDir(), "noop")
	res := r.Apply(context.Background(), sid, []RunnableTask{Task{
		ID: "dry", Exec: Phase{{Command: "touch " + target}},
	}})
	if res[0].Status != store.TaskSkipped {
		t.Fatalf("dry-run should skip, got %s", res[0].Status)
	}
	if _, err := os.Stat(target); err == nil {
		t.Fatal("dry-run created file")
	}
}

func TestApply_DryRunDoesNotStoreBackupData(t *testing.T) {
	r, _ := newRunner(t)
	r.DryRun = true
	backupFile := filepath.Join(t.TempDir(), "cfg")
	writeFile(t, backupFile, "SECRET")
	sid, _ := r.StartSession()

	_ = r.Apply(context.Background(), sid, []RunnableTask{Task{
		ID:          "dry_backup",
		BackupFiles: []string{backupFile},
		Exec:        Phase{{Command: "true"}},
	}})

	execs, err := r.Store.TasksForSession(sid)
	if err != nil {
		t.Fatal(err)
	}
	if len(execs) != 1 {
		t.Fatalf("expected one execution, got %d", len(execs))
	}
	if len(execs[0].BackupData) != 0 || execs[0].BackupPath != "" {
		t.Fatalf("dry-run stored backup data: %+v", execs[0])
	}
}

func TestApply_PreCheckMismatchSkipsForAnyExpectedExit(t *testing.T) {
	r, _ := newRunner(t)
	target := filepath.Join(t.TempDir(), "noop")
	sid, _ := r.StartSession()

	res := r.Apply(context.Background(), sid, []RunnableTask{Task{
		ID:       "already_done",
		PreCheck: PreCheck{Command: "true", ExpectExit: 1},
		Exec:     Phase{{Command: "touch " + target}},
	}})

	if res[0].Status != store.TaskSkipped {
		t.Fatalf("expected skipped, got %s", res[0].Status)
	}
	if _, err := os.Stat(target); err == nil {
		t.Fatal("pre-check mismatch should not execute task")
	}
}

func TestRollbackSessionTasksSkipsSkippedExecutions(t *testing.T) {
	r, _ := newRunner(t)
	backupFile := filepath.Join(t.TempDir(), "cfg")
	writeFile(t, backupFile, "CURRENT")
	sid, _ := r.StartSession()

	if err := r.Store.RecordTask(store.TaskExecution{
		ID:               "exec-1",
		SessionID:        sid,
		TaskID:           "skip_me",
		Status:           store.TaskSkipped,
		RollbackPossible: true,
		BackupPath:       backupFile,
		BackupData:       []byte("OLD"),
		ExecutedAt:       time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	results, err := r.RollbackSessionTasks(context.Background(), sid, []RunnableTask{Task{
		ID:               "skip_me",
		RollbackPossible: true,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Status != store.TaskSkipped {
		t.Fatalf("results = %+v", results)
	}
	got, _ := os.ReadFile(backupFile)
	if string(got) != "CURRENT" {
		t.Fatalf("skipped execution was rolled back, content = %q", got)
	}
}
