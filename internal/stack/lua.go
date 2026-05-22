package stack

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

// RunHook executes a named function (install/update/backup/status) from a stack.lua file.
// If the function does not exist the call is a no-op (graceful).
// log is called for every ctx.log(msg) call made from Lua.
func RunHook(luaPath, funcName, stackDir string, log func(string)) error {
	L := lua.NewState()
	defer L.Close()

	// Provide a context table: ctx.log, ctx.dir
	ctx := L.NewTable()
	L.SetField(ctx, "dir", lua.LString(stackDir))
	L.SetField(ctx, "log", L.NewFunction(func(L *lua.LState) int {
		msg := L.CheckString(1)
		log("[lua] " + msg)
		return 0
	}))
	L.SetGlobal("_ctx", ctx)

	if err := L.DoFile(luaPath); err != nil {
		return fmt.Errorf("load %s: %w", luaPath, err)
	}

	fn := L.GetGlobal(funcName)
	if fn == lua.LNil {
		// Function not defined — that's fine.
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
