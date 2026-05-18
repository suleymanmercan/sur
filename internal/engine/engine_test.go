package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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
	results := r.Apply(context.Background(), sid, []Task{{
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
	res := r.Apply(context.Background(), sid, []Task{task})
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
	res := r.Apply(context.Background(), sid, []Task{{
		ID: "dry", Exec: Phase{{Command: "touch " + target}},
	}})
	if res[0].Status != store.TaskSkipped {
		t.Fatalf("dry-run should skip, got %s", res[0].Status)
	}
	if _, err := os.Stat(target); err == nil {
		t.Fatal("dry-run created file")
	}
}
