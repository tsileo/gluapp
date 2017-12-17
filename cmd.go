package gluapp

import (
	"os/exec"
	"strings"

	"github.com/yuin/gopher-lua"
)

func setupCmd(cwd string) func(*lua.LState) int {
	return func(L *lua.LState) int {
		// register functions to the table
		mod := L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
			"run": func(L *lua.LState) int {
				parts := strings.Split(L.ToString(1), " ")
				cmd := exec.Command(parts[0], parts[1:]...)
				cmd.Dir = cwd
				err := cmd.Run()
				var out string
				if err != nil {
					out = err.Error()
				}
				L.Push(lua.LString(out))
				return 1
			},
		})
		// returns the module
		L.Push(mod)
		return 1
	}
}
