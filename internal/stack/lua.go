package stack

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// RunHook executes a named function (install/update/backup/status/rotate) from a
// stack.lua file. If the function does not exist the call is a no-op (graceful).
//
// Lua context API exposed via the `ctx` table:
//
//	ctx.dir               — absolute path of the installed stack directory
//	ctx.log(msg)          — write a line to the TUI output
//	ctx.exec(cmd, ...)    — run a command; streams stdout+stderr to ctx.log;
//	                         returns (ok bool, output string)
//	ctx.read_secret(name) — read ./secrets/<name>.txt; returns string or ""
//	ctx.write_secret(name, value) — overwrite ./secrets/<name>.txt (mode 0600)
//	ctx.env(key)          — read an environment variable (e.g. SUDO_USER)
func RunHook(luaPath, funcName, stackDir string, log func(string)) error {
	L := lua.NewState()
	defer L.Close()

	ctx := L.NewTable()
	L.SetField(ctx, "dir", lua.LString(stackDir))

	// ctx.log(msg)
	L.SetField(ctx, "log", L.NewFunction(func(L *lua.LState) int {
		msg := L.CheckString(1)
		log("[lua] " + msg)
		return 0
	}))

	// ctx.exec(cmd, arg1, arg2, ...) → (ok, combined_output)
	L.SetField(ctx, "exec", L.NewFunction(func(L *lua.LState) int {
		n := L.GetTop()
		if n < 1 {
			L.Push(lua.LFalse)
			L.Push(lua.LString("exec: no command given"))
			return 2
		}
		cmdName := L.CheckString(1)
		var args []string
		for i := 2; i <= n; i++ {
			args = append(args, L.CheckString(i))
		}
		log(fmt.Sprintf("[exec] %s %s", cmdName, strings.Join(args, " ")))
		cmd := exec.Command(cmdName, args...) // #nosec G204
		cmd.Env = os.Environ()
		out, err := cmd.CombinedOutput()
		outStr := strings.TrimSpace(string(out))
		for _, line := range strings.Split(outStr, "\n") {
			if strings.TrimSpace(line) != "" {
				log("  " + line)
			}
		}
		if err != nil {
			L.Push(lua.LFalse)
			L.Push(lua.LString(outStr))
			return 2
		}
		L.Push(lua.LTrue)
		L.Push(lua.LString(outStr))
		return 2
	}))

	// ctx.read_secret(name) → string  (reads ./secrets/<name>.txt)
	L.SetField(ctx, "read_secret", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		p := filepath.Join(stackDir, "secrets", name+".txt")
		b, err := os.ReadFile(p) // #nosec G304
		if err != nil {
			L.Push(lua.LString(""))
			return 1
		}
		L.Push(lua.LString(strings.TrimSpace(string(b))))
		return 1
	}))

	// ctx.write_secret(name, value) → ok bool
	L.SetField(ctx, "write_secret", L.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		value := L.CheckString(2)
		p := filepath.Join(stackDir, "secrets", name+".txt")
		err := os.WriteFile(p, []byte(value), 0o600) // #nosec G306
		if err != nil {
			log(fmt.Sprintf("[lua] write_secret error: %v", err))
			L.Push(lua.LFalse)
			return 1
		}
		L.Push(lua.LTrue)
		return 1
	}))

	// ctx.env(key) → string
	L.SetField(ctx, "env", L.NewFunction(func(L *lua.LState) int {
		key := L.CheckString(1)
		L.Push(lua.LString(os.Getenv(key)))
		return 1
	}))

	L.SetGlobal("_ctx", ctx)

	if err := L.DoFile(luaPath); err != nil {
		return fmt.Errorf("load %s: %w", luaPath, err)
	}

	fn := L.GetGlobal(funcName)
	if fn == lua.LNil {
		return nil
	}

	if err := L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    0,
		Protect: true,
	}, ctx); err != nil {
		return fmt.Errorf("hook %s: %w", funcName, err)
	}
	return nil
}
