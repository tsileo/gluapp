package gluapp

import (
	"bytes"
	"html/template"

	"github.com/yuin/gopher-lua"
)

// template.Must(template.ParseGlob("YOURTEMPLATEDIR/*"))
func setupTemplate() func(*lua.LState) int {
	return func(L *lua.LState) int {
		// Setup the router module
		mod := L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
			"render_string": func(L *lua.LState) int {
				var out bytes.Buffer
				tpl, err := template.New("").Parse(L.ToString(1))
				if err != nil {
					// TODO(tsileo): return error?
					return 0
				}
				if err := tpl.Execute(&out, tableToMap(L.ToTable(2))); err != nil {
					return 0
				}
				L.Push(lua.LString(out.String()))
				return 1
			},
		})
		L.Push(mod)
		return 1
	}
}
