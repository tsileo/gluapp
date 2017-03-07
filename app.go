package gluapp

import (
	"github.com/yuin/gopher-lua"
)

// App represents a Lua app
type App struct {
	ls          *lua.LState
	conf        *Config
	publicIndex map[string]struct{}
}
