package engine

import (
	"context"
	"embed"
	"fmt"
	"os"

	lua "github.com/yuin/gopher-lua"
)

// LuaTask represents a task defined via a Lua script.
type LuaTask struct {
	ID               string
	Name             string
	Description      string
	RollbackPossible bool
	BackupFiles      []string
	RiskLevel        string
	Distros          []string
	ScriptPath       string
	ScriptContent    string
}

// GetID returns the task ID.
func (t LuaTask) GetID() string { return t.ID }

// GetName returns the task name.
func (t LuaTask) GetName() string { return t.Name }

// GetDescription returns the task description.
func (t LuaTask) GetDescription() string { return t.Description }

// GetRollbackPossible returns rollback capability.
func (t LuaTask) GetRollbackPossible() bool { return t.RollbackPossible }

// GetBackupFiles returns files to backup.
func (t LuaTask) GetBackupFiles() []string { return t.BackupFiles }

// GetRiskLevel returns the task risk level.
func (t LuaTask) GetRiskLevel() string { return t.RiskLevel }

// GetDistros returns supported distributions.
func (t LuaTask) GetDistros() []string { return t.Distros }

// NeedsRun calls pre_check() inside the Lua script.
func (t LuaTask) NeedsRun(ctx context.Context) (bool, int) {
	L := lua.NewState()
	defer L.Close()

	registerBridge(ctx, L, nil, nil)

	if err := L.DoString(t.ScriptContent); err != nil {
		return true, -1
	}

	preCheckFn := L.GetGlobal("pre_check")
	if preCheckFn.Type() != lua.LTFunction {
		return true, 0
	}

	err := L.CallByParam(lua.P{
		Fn:      preCheckFn,
		NRet:    2,
		Protect: true,
	})
	if err != nil {
		return true, -1
	}

	retVal := L.Get(-2)
	codeVal := L.Get(-1)
	L.Pop(2)

	needsRun := true
	if retVal.Type() == lua.LTBool {
		needsRun = lua.LVAsBool(retVal)
	}

	code := 0
	if codeVal.Type() == lua.LTNumber {
		code = int(lua.LVAsNumber(codeVal))
	}

	return needsRun, code
}

// Execute calls exec() inside the Lua script.
func (t LuaTask) Execute(ctx context.Context, dryRun bool, logFn func(format string, args ...any)) error {
	if dryRun {
		logFn("[%s] DRY-RUN — running Lua exec() in dry-run mode (read-only)", t.ID)
	}

	L := lua.NewState()
	defer L.Close()

	registerBridge(ctx, L, logFn, &dryRun)

	if err := L.DoString(t.ScriptContent); err != nil {
		return err
	}

	execFn := L.GetGlobal("exec")
	if execFn.Type() != lua.LTFunction {
		return fmt.Errorf("task 'exec' function is required")
	}

	err := L.CallByParam(lua.P{
		Fn:      execFn,
		NRet:    1,
		Protect: true,
	})
	if err != nil {
		return err
	}

	retVal := L.Get(-1)
	L.Pop(1)

	if retVal != lua.LNil && retVal.Type() == lua.LTString {
		return fmt.Errorf("%s", retVal.String())
	}

	return nil
}

// Verify calls post_check() inside the Lua script.
func (t LuaTask) Verify(ctx context.Context) error {
	L := lua.NewState()
	defer L.Close()

	registerBridge(ctx, L, nil, nil)

	if err := L.DoString(t.ScriptContent); err != nil {
		return err
	}

	postCheckFn := L.GetGlobal("post_check")
	if postCheckFn.Type() != lua.LTFunction {
		return nil
	}

	err := L.CallByParam(lua.P{
		Fn:      postCheckFn,
		NRet:    1,
		Protect: true,
	})
	if err != nil {
		return err
	}

	retVal := L.Get(-1)
	L.Pop(1)

	if retVal != lua.LNil && retVal.Type() == lua.LTString {
		return fmt.Errorf("post_check: %s", retVal.String())
	}

	return nil
}

// Revert calls rollback() inside the Lua script.
func (t LuaTask) Revert(ctx context.Context, backupPath string, logFn func(format string, args ...any)) error {
	L := lua.NewState()
	defer L.Close()

	registerBridge(ctx, L, logFn, nil)

	if err := L.DoString(t.ScriptContent); err != nil {
		return err
	}

	rollbackFn := L.GetGlobal("rollback")
	if rollbackFn.Type() != lua.LTFunction {
		return nil
	}

	err := L.CallByParam(lua.P{
		Fn:      rollbackFn,
		NRet:    1,
		Protect: true,
	}, lua.LString(backupPath))
	if err != nil {
		return err
	}

	retVal := L.Get(-1)
	L.Pop(1)

	if retVal != lua.LNil && retVal.Type() == lua.LTString {
		return fmt.Errorf("rollback: %s", retVal.String())
	}

	return nil
}

// LoadLuaTask reads and parses a Lua task from a file path.
func LoadLuaTask(path string) (LuaTask, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return LuaTask{}, err
	}
	return ParseLuaTask(string(b), path)
}

// LoadLuaTaskFS reads and parses a Lua task from an embedded FS.
func LoadLuaTaskFS(fs embed.FS, path string) (LuaTask, error) {
	b, err := fs.ReadFile(path)
	if err != nil {
		return LuaTask{}, err
	}
	return ParseLuaTask(string(b), path)
}

// ParseLuaTask parses task metadata defined at the global scope in Lua.
func ParseLuaTask(script, path string) (LuaTask, error) {
	L := lua.NewState()
	defer L.Close()

	if err := L.DoString(script); err != nil {
		return LuaTask{}, err
	}

	id := L.GetGlobal("id")
	if id.Type() != lua.LTString {
		return LuaTask{}, fmt.Errorf("task 'id' must be a string")
	}

	name := L.GetGlobal("name")
	nameStr := ""
	if name.Type() == lua.LTString {
		nameStr = name.String()
	}

	desc := L.GetGlobal("description")
	descStr := ""
	if desc.Type() == lua.LTString {
		descStr = desc.String()
	}

	rollbackPossible := L.GetGlobal("rollback_possible")
	rbPossible := false
	if rollbackPossible.Type() == lua.LTBool {
		rbPossible = lua.LVAsBool(rollbackPossible)
	}

	risk := L.GetGlobal("risk_level")
	riskStr := "low"
	if risk.Type() == lua.LTString {
		riskStr = risk.String()
	}

	var distros []string
	distrosVal := L.GetGlobal("distros")
	if tbl, ok := distrosVal.(*lua.LTable); ok {
		tbl.ForEach(func(_, v lua.LValue) {
			if v.Type() == lua.LTString {
				distros = append(distros, v.String())
			}
		})
	}

	var backupFiles []string
	backupVal := L.GetGlobal("backup_files")
	if tbl, ok := backupVal.(*lua.LTable); ok {
		tbl.ForEach(func(_, v lua.LValue) {
			if v.Type() == lua.LTString {
				backupFiles = append(backupFiles, v.String())
			}
		})
	}

	return LuaTask{
		ID:               id.String(),
		Name:             nameStr,
		Description:      descStr,
		RollbackPossible: rbPossible,
		BackupFiles:      backupFiles,
		RiskLevel:        riskStr,
		Distros:          distros,
		ScriptPath:       path,
		ScriptContent:    script,
	}, nil
}

func registerBridge(ctx context.Context, L *lua.LState, logFn func(format string, args ...any), dryRun *bool) {
	L.SetGlobal("run", L.NewFunction(func(L *lua.LState) int {
		cmd := L.CheckString(1)
		if dryRun != nil && *dryRun {
			if logFn != nil {
				logFn("       $ (dry-run) %s", cmd)
			}
			L.Push(lua.LString(""))
			L.Push(lua.LNumber(0))
			return 2
		}
		out, code := runShell(ctx, cmd)
		L.Push(lua.LString(out))
		L.Push(lua.LNumber(code))
		return 2
	}))

	L.SetGlobal("log", L.NewFunction(func(L *lua.LState) int {
		msg := L.CheckString(1)
		if logFn != nil {
			logFn("       > %s", msg)
		}
		return 0
	}))

	L.SetGlobal("read_file", L.NewFunction(func(L *lua.LState) int {
		path := L.CheckString(1)
		b, err := os.ReadFile(path)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		L.Push(lua.LString(string(b)))
		L.Push(lua.LNil)
		return 2
	}))

	L.SetGlobal("write_file", L.NewFunction(func(L *lua.LState) int {
		path := L.CheckString(1)
		content := L.CheckString(2)
		if dryRun != nil && *dryRun {
			if logFn != nil {
				logFn("       $ (dry-run) write %d bytes to %s", len(content), path)
			}
			L.Push(lua.LNil)
			return 1
		}
		err := os.WriteFile(path, []byte(content), 0o644)
		if err != nil {
			L.Push(lua.LString(err.Error()))
			return 1
		}
		L.Push(lua.LNil)
		return 1
	}))

	L.SetGlobal("file_exists", L.NewFunction(func(L *lua.LState) int {
		path := L.CheckString(1)
		_, err := os.Stat(path)
		L.Push(lua.LBool(err == nil))
		return 1
	}))
}
