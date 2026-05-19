package engine

import (
	"context"
	"fmt"
	"os"
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

func TestLuaTask_SyntaxError(t *testing.T) {
	script := `
id = "syntax_error_task"
name = "Syntax Error Task"
description = "This script is broken"
distros = { "debian" }

function pre_check()
    -- Missing end keyword to trigger syntax error
    if true then
        return true
`
	_, err := ParseLuaTask(script, "broken.lua")
	if err == nil {
		t.Fatal("expected parsing error due to syntax error, got nil")
	}
}

func TestLuaTask_MissingCallbacks(t *testing.T) {
	// A Lua task only requires ID. Missing lifecycle callbacks should fallback gracefully.
	script := `
id = "minimal_task"
name = "Minimal Lua Task"
`
	task, err := ParseLuaTask(script, "minimal.lua")
	if err != nil {
		t.Fatalf("failed to parse minimal task: %v", err)
	}

	ctx := context.Background()

	// 1. Missing pre_check should default to needsRun=true, code=0
	needsRun, code := task.NeedsRun(ctx)
	if !needsRun || code != 0 {
		t.Errorf("expected pre_check default NeedsRun=true, code=0; got needsRun=%v, code=%d", needsRun, code)
	}

	// 2. Missing exec should fail since it's required in Execute
	err = task.Execute(ctx, false, nil)
	if err == nil {
		t.Error("expected execute to fail because exec function is required, got nil")
	}

	// 3. Missing post_check should default to success (nil)
	err = task.Verify(ctx)
	if err != nil {
		t.Errorf("expected verify to succeed when post_check is missing, got: %v", err)
	}

	// 4. Missing rollback should default to success (nil)
	err = task.Revert(ctx, "/tmp/backup", nil)
	if err != nil {
		t.Errorf("expected revert to succeed when rollback is missing, got: %v", err)
	}
}

func TestLuaTask_SingleReturnValuePreCheck(t *testing.T) {
	// pre_check returns single values (e.g. true or false) or nothing
	t.Run("returns only boolean true", func(t *testing.T) {
		script := `
id = "single_return_true"
function pre_check()
    return true
end
`
		task, err := ParseLuaTask(script, "test.lua")
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		needsRun, code := task.NeedsRun(context.Background())
		if !needsRun || code != 0 {
			t.Errorf("expected needsRun=true, code=0; got needsRun=%v, code=%d", needsRun, code)
		}
	})

	t.Run("returns only boolean false", func(t *testing.T) {
		script := `
id = "single_return_false"
function pre_check()
    return false
end
`
		task, err := ParseLuaTask(script, "test.lua")
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		needsRun, code := task.NeedsRun(context.Background())
		if needsRun || code != 0 {
			t.Errorf("expected needsRun=false, code=0; got needsRun=%v, code=%d", needsRun, code)
		}
	})

	t.Run("returns nothing", func(t *testing.T) {
		script := `
id = "returns_nothing"
function pre_check()
    -- no return statement
end
`
		task, err := ParseLuaTask(script, "test.lua")
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		needsRun, code := task.NeedsRun(context.Background())
		// Defaults to needsRun=true, code=0
		if !needsRun || code != 0 {
			t.Errorf("expected needsRun=true, code=0; got needsRun=%v, code=%d", needsRun, code)
		}
	})
}

func TestLuaTask_BridgeAPI(t *testing.T) {
	// Create a temporary file to test file operations
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_file.txt")

	script := fmt.Sprintf(`
id = "bridge_test_task"
name = "Bridge Test"
rollback_possible = false

function exec()
    -- Test log
    log("Testing write_file...")
    
    -- Test write_file
    local write_err = write_file("%s", "hello from lua")
    if write_err ~= nil then
        return "write_file failed: " .. write_err
    end

    -- Test file_exists
    if not file_exists("%s") then
        return "file does not exist after write"
    end

    -- Test read_file
    local content, read_err = read_file("%s")
    if read_err ~= nil then
        return "read_file failed: " .. read_err
    end
    if content ~= "hello from lua" then
        return "unexpected file content: " .. content
    end

    -- Test run command (echo)
    local out, code = run("echo 'shell execution works'")
    if code ~= 0 then
        return "run failed with code " .. code
    end
    log("shell out: " .. out)

    return nil
end
`, testFilePath, testFilePath, testFilePath)

	task, err := ParseLuaTask(script, "bridge_test.lua")
	if err != nil {
		t.Fatalf("failed to parse task: %v", err)
	}

	var logged []string
	logFn := func(format string, args ...any) {
		logged = append(logged, fmt.Sprintf(format, args...))
	}

	// 1. Run actual execution
	err = task.Execute(context.Background(), false, logFn)
	if err != nil {
		t.Errorf("expected execute success, got error: %v", err)
	}

	// Verify file was written
	if !fileExists(testFilePath) {
		t.Errorf("expected file %s to be created on disk", testFilePath)
	}

	// 2. Run in dry-run mode (should log and not write/run commands)
	dryRunFilePath := filepath.Join(tempDir, "dry_run_file.txt")
	scriptDryRun := fmt.Sprintf(`
id = "dry_run_test"
name = "Dry Run Test"

function exec()
    local write_err = write_file("%s", "dry run content")
    if write_err ~= nil then
        return write_err
    end
    run("touch %s")
    return nil
end
`, dryRunFilePath, dryRunFilePath)

	taskDryRun, err := ParseLuaTask(scriptDryRun, "dry_run.lua")
	if err != nil {
		t.Fatalf("failed to parse dry-run task: %v", err)
	}

	logged = nil
	err = taskDryRun.Execute(context.Background(), true, logFn)
	if err != nil {
		t.Errorf("expected dry-run execute success, got error: %v", err)
	}

	// The file should NOT have been created because dry_run=true bypasses write_file and run
	if fileExists(dryRunFilePath) {
		t.Errorf("expected file %s to NOT be created in dry-run mode", dryRunFilePath)
	}
}

// Helper function to check file existence in tests
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
