package gluapp

import (
	"testing"

	"github.com/yuin/gopher-lua"
)

func TestTemplate(t *testing.T) {
	// Create a new empty state
	L := lua.NewState()
	defer L.Close()

	// Setup the state
	L.PreloadModule("template", setupTemplate())
	setupTestState(L, t)

	// Execute the Lua code
	if err := L.DoString(`
tpl = require('template')
logf('testing template')
out = tpl.render_string('Hello {{.world}} from template', {world = 'World'})
expected = 'Hello World from template'
if out ~= expected then
  errorf('render_string error, got %v, expected %v', out, expected)
end
`); err != nil {
		panic(err)
	}
}
