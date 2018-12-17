package gluapp

import (
	"bytes"
	"html/template"
	"path/filepath"

	"a4.io/blobstash/pkg/apps/luautil"

	"github.com/gomarkdown/markdown"
	"github.com/yuin/gopher-lua"
)

var funcs = template.FuncMap{
	"markdownify": func(raw interface{}) template.HTML {
		switch md := raw.(type) {
		case string:
			return template.HTML(markdown.ToHTML([]byte(md), nil, nil))
		case lua.LString:
			return template.HTML(markdown.ToHTML([]byte(string(md)), nil, nil))
		default:
			panic("bad md type")
		}
	},
}

func setupTemplate(path string) func(*lua.LState) int {
	return func(L *lua.LState) int {
		// Setup the router module
		mod := L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
			"render_string": func(L *lua.LState) int {
				var out bytes.Buffer
				tpl, err := template.New("").Funcs(funcs).Parse(L.ToString(1))
				if err != nil {
					// TODO(tsileo): return error?
					return 0
				}
				if err := tpl.Execute(&out, luautil.TableToMap(L.ToTable(2))); err != nil {
					L.Push(lua.LString(err.Error()))
					return 1
				}
				L.Push(lua.LString(out.String()))
				return 1
			},
			"render": func(L *lua.LState) int {
				var out bytes.Buffer

				var templates []string
				for i := 1; i < L.GetTop(); i++ {
					templates = append(templates, filepath.Join(path, string(L.ToString(i))))
				}

				tmpl, err := template.New("").Funcs(funcs).ParseFiles(templates...)
				if err != nil {
					L.Push(lua.LString(err.Error()))
					return 1
				}
				tmplName := filepath.Base(templates[len(templates)-1])
				ctx := luautil.TableToMap(L.ToTable(L.GetTop()))
				if err := tmpl.ExecuteTemplate(&out, tmplName, ctx); err != nil {
					L.Push(lua.LString(err.Error()))
					return 1
				}

				L.Push(lua.LString(out.String()))
				return 1
			},
		})
		L.Push(mod)
		return 1
	}
}
