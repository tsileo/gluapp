/*

The Router implements a basic HTTP router that does not rely on `net/http` at all.

It supports named parameters (`/hello/:name`), insertion order of routes does matter, the first matching route is
returned.

Designed to be used as router for luareq.

*/
package gluapp

import (
	"fmt"
	"regexp"
	"strings"

	"a4.io/blobstash/pkg/apps/luautil" // FIXME(tsileo): copy luatuil
	"github.com/yuin/gopher-lua"
)

const any = "any"

// route represents a registed route method/path
type route struct {
	path   string
	method string
	regexp *regexp.Regexp
	data   interface{}
}

// params represents a route named parameters
type params map[string]string

func (r *route) match(method, path string) (bool, params) {
	if method != any && method != r.method {
		return false, nil
	}
	if r.regexp != nil {
		matches := r.regexp.FindStringSubmatch(path)
		if matches != nil {
			params := params{}
			for i, k := range r.regexp.SubexpNames()[1:] {
				params[k] = matches[i+1]
			}
			return true, params
		}
	}
	if path == r.path {
		return true, nil
	}
	return false, nil
}

// TODO(tsileo): use the router directly and embed a request in it, also use LFunction instead of interface{}
// Router represents the router and holds the routes.
type router struct {
	method, path string
	routes       []*route
}

// TODO(tsileo): return method not authorized

func setupRouter(method, path string) func(*lua.LState) int {
	return func(L *lua.LState) int {
		mod := L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
			"new": func(L *lua.LState) int {
				mt := L.NewTypeMetatable("router")
				// methods
				routerMethods := map[string]lua.LGFunction{
					"any": routerMethodFunc(any),
					"run": routerRun,
				}
				for _, m := range methods {
					routerMethods[strings.ToLower(m)] = routerMethodFunc(m)
				}
				L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), routerMethods))
				router := &router{routes: []*route{}, method: method, path: path}
				ud := L.NewUserData()
				ud.Value = router
				L.SetMetatable(ud, L.GetTypeMetatable("router"))
				L.Push(ud)
				return 1
			},
		})
		L.Push(mod)
		return 1
	}
}

func checkRouter(L *lua.LState) *router {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*router); ok {
		return v
	}
	L.ArgError(1, "router expected")
	return nil
}

func routerMethodFunc(method string) func(*lua.LState) int {
	return func(L *lua.LState) int {
		router := checkRouter(L)
		if router == nil {
			return 1
		}
		path := string(L.CheckString(2))
		fn := L.CheckFunction(3)
		if method == "any" {
			for _, m := range methods {
				router.add(m, path, fn)
			}

		} else {
			router.add(method, path, fn)
		}
		return 0
	}
}

func routerRun(L *lua.LState) int {
	router := checkRouter(L)
	if router == nil {
		return 1
	}
	fn, params := router.match(router.method, router.path)
	p := map[string]interface{}{}
	for k, v := range params {
		p[k] = v
	}
	if err := L.CallByParam(lua.P{
		Fn:      lua.LValue(fn.(*lua.LFunction)),
		NRet:    0,
		Protect: true,
	}, luautil.InterfaceToLValue(L, p)); err != nil {
		panic(err)
	}
	return 0
}

// Add adds the path to the router, order of insertions matters as the first matched route is returned.
func (r *router) add(method, path string, data interface{}) {
	// TODO(tsileo): make more verification on the path?
	newRoute := &route{
		data:   data,
		path:   path,
		method: method,
	}
	if strings.Contains(path, ":") {
		newRoute.path = ""
		parts := strings.Split(path, "/")
		var rparts []string
		var hasRegexp bool
		for _, part := range parts {
			// Check if the key is a named parameters
			if strings.HasPrefix(part, ":") && strings.Contains(part, ":") {
				hasRegexp = true
				rparts = append(rparts, fmt.Sprintf("(?P<%s>[^/]+)", part[1:]))
			} else {
				rparts = append(rparts, part)
			}
		}
		// Ensure a regex is needed
		if hasRegexp {
			sreg := strings.Join(rparts, "/")
			reg := regexp.MustCompile(sreg)
			newRoute.regexp = reg

		} else {
			// Fallback to basic string matching
			newRoute.path = path
		}
	}
	r.routes = append(r.routes, newRoute)
}

// Match returns the given route data alog with the params if any matches
func (r *router) match(method, path string) (interface{}, params) {
	for _, rt := range r.routes {
		match, params := rt.match(method, path)
		if match {
			return rt.data, params
		}
	}
	return nil, nil
}
