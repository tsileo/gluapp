package gluapp

import (
	"github.com/yuin/gopher-lua"
)

func loadHTTP(L *lua.LState) int {
	// register functions to the table
	mod := L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{})
	// returns the module
	L.Push(mod)
	return 1
}
