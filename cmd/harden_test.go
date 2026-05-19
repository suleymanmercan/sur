package cmd

import (
	"testing"

	"github.com/suleymanmercan/sur/internal/engine"
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
