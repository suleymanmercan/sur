package cmd

import (
	"embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/suleymanmercan/sur/internal/engine"
	"github.com/suleymanmercan/sur/internal/osdetect"
)

func TestFilterTasksUnknownID(t *testing.T) {
	_, err := filterTasks([]engine.RunnableTask{engine.Task{ID: "known"}}, []string{"known", "missing"})
	if err == nil {
		t.Fatal("expected unknown task id error")
	}
}

func TestFilterTasksOnlyKnown(t *testing.T) {
	got, err := filterTasks([]engine.RunnableTask{engine.Task{ID: "a"}, engine.Task{ID: "b"}}, []string{"b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].GetID() != "b" {
		t.Fatalf("filtered tasks = %+v", got)
	}
}

func TestSupportsOSMatchesDistroID(t *testing.T) {
	task := engine.Task{Distros: []string{"ubuntu", "debian"}}
	info := &osdetect.OSInfo{ID: "debian", Family: osdetect.FamilyDebian}
	if !supportsOS(task, info) {
		t.Fatal("expected Debian task support")
	}
}

func TestSupportsOSRejectsDifferentDistro(t *testing.T) {
	task := engine.Task{Distros: []string{"ubuntu", "debian"}}
	info := &osdetect.OSInfo{ID: "fedora", Family: osdetect.FamilyFedora}
	if supportsOS(task, info) {
		t.Fatal("expected Fedora to be rejected for Debian-only task")
	}
}

func TestSupportsOSNormalizesAlmaAlias(t *testing.T) {
	task := engine.Task{Distros: []string{"alma"}}
	info := &osdetect.OSInfo{ID: "almalinux", Family: osdetect.FamilyRHEL}
	if !supportsOS(task, info) {
		t.Fatal("expected alma alias to match almalinux")
	}
}

func TestLoadTaskSetMerging(t *testing.T) {
	tempDir := t.TempDir()

	overrideTaskPath := filepath.Join(tempDir, "custom_task.yaml")
	err := os.WriteFile(overrideTaskPath, []byte(`
id: custom_test_task
name: Custom Test Task
`), 0644)
	if err != nil {
		t.Fatalf("failed to write override task: %v", err)
	}

	var emptyFS embed.FS
	tasks, err := loadTaskSet(emptyFS, "tasks", tempDir)
	if err != nil {
		t.Fatalf("loadTaskSet failed: %v", err)
	}

	found := false
	for _, task := range tasks {
		if task.GetID() == "custom_test_task" {
			found = true
			if task.GetName() != "Custom Test Task" {
				t.Errorf("expected task name 'Custom Test Task', got %q", task.GetName())
			}
		}
	}
	if !found {
		t.Error("expected custom_test_task to be loaded from overrideDir")
	}
}
