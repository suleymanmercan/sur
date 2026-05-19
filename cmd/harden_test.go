package cmd

import (
	"testing"

	"github.com/suleymanmercan/sur/internal/engine"
	"github.com/suleymanmercan/sur/internal/osdetect"
)

func TestFilterTasksUnknownID(t *testing.T) {
	_, err := filterTasks([]engine.Task{{ID: "known"}}, []string{"known", "missing"})
	if err == nil {
		t.Fatal("expected unknown task id error")
	}
}

func TestFilterTasksOnlyKnown(t *testing.T) {
	got, err := filterTasks([]engine.Task{{ID: "a"}, {ID: "b"}}, []string{"b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "b" {
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
