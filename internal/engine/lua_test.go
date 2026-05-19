package engine

import (
	"context"
	"path/filepath"
	"testing"
)

func TestLuaTask_ExecutionAndPreCheck(t *testing.T) {
	script := `
id = "test_lua_task"
name = "Test Lua Task"
description = "Verifies Lua task execution"
rollback_possible = true
backup_files = { "/tmp/nonexistent_file_xyz" }
risk_level = "low"
distros = { "debian", "ubuntu" }

function pre_check()
    -- check if file exists using standard API
    if file_exists("/tmp/nonexistent_file_xyz") then
        return false, 0
    end
    return true, 0
end

function exec()
    log("Running exec function")
    return nil
end

function post_check()
    log("Running post check")
    return nil
end

function rollback(backup_path)
    log("Rolling back " .. backup_path)
    return nil
end
`
	task, err := ParseLuaTask(script, "test.lua")
	if err != nil {
		t.Fatalf("failed to parse lua task: %v", err)
	}

	if task.GetID() != "test_lua_task" {
		t.Errorf("expected id 'test_lua_task', got %q", task.GetID())
	}
	if task.GetName() != "Test Lua Task" {
		t.Errorf("expected name 'Test Lua Task', got %q", task.GetName())
	}
	if !task.GetRollbackPossible() {
		t.Errorf("expected rollback possible true")
	}

	ctx := context.Background()

	// 1. Run Pre-Check
	needsRun, code := task.NeedsRun(ctx)
	if !needsRun || code != 0 {
		t.Errorf("expected needsRun=true, code=0; got needsRun=%v, code=%d", needsRun, code)
	}

	// 2. Run Execute
	var logged []string
	logFn := func(format string, args ...any) {
		logged = append(logged, format)
	}
	err = task.Execute(ctx, false, logFn)
	if err != nil {
		t.Errorf("expected execute success, got: %v", err)
	}

	// 3. Run Verify (PostCheck)
	err = task.Verify(ctx)
	if err != nil {
		t.Errorf("expected verify success, got: %v", err)
	}

	// 4. Run Revert (Rollback)
	err = task.Revert(ctx, "/tmp/backup", logFn)
	if err != nil {
		t.Errorf("expected revert success, got: %v", err)
	}
}

func TestLoadAllRunnableTasks(t *testing.T) {
	dir := t.TempDir()

	// Write a YAML task
	writeFile(t, filepath.Join(dir, "task1.yaml"), `
id: yaml_task
name: YAML Task
`)

	// Write a Lua task
	writeFile(t, filepath.Join(dir, "task2.lua"), `
id = "lua_task"
name = "Lua Task"
`)

	tasks, err := LoadAllRunnableTasks(dir)
	if err != nil {
		t.Fatalf("failed to load all tasks: %v", err)
	}

	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	// Tasks should be sorted by ID: "lua_task", "yaml_task"
	if tasks[0].GetID() != "lua_task" || tasks[1].GetID() != "yaml_task" {
		t.Errorf("unexpected task sorting or IDs: %s, %s", tasks[0].GetID(), tasks[1].GetID())
	}
}
