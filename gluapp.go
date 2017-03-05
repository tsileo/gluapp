package gluapp // import "a4.io/gluapp"

import (
	"net/http"

	"github.com/yuin/gopher-lua"
)

// TODO(tsileo): move request and response to their own file
// TODO(tsileo): config for the path
// TODO(tsileo): render go template via the path
// TODO(tsileo): render public dir via whitelist, then execute the Lua app
// TODO(tsileo): a logFunc(t time.Time, msg string, args ...interface{})?
// TODO(tsileo): `body, resp, err = http:get('http://...')`
// `body:json()`, resp.statuscode

var methods = []string{
	"GET", "POST", "PUT", "PATCH", "DELETE", "TRACE", "CONNECT", "OPTIONS", "HEAD",
}

// Exec run the code as a Lua script
func Exec(code string, w http.ResponseWriter, r *http.Request) error {
	// TODO(tsileo): clean error, take L as argument
	L := lua.NewState()
	defer L.Close()

	// Update the path if needed
	// FIXME(tsileo): handle the path via config
	// path, ok := L.GetField(L.GetField(L.Get(lua.EnvironIndex), "package"), "path").(lua.LString)
	// if ok {
	// 	fmt.Printf("path=%s\n", path)
	// }
	// path = "/Users/thomas/gopath/src/github.com/tsileo/?.lua;" + path
	// L.SetField(L.GetField(L.Get(lua.EnvironIndex), "package"), "path", lua.LString(path))

	// Setup `request`
	if err := setupRequest(L, r); err != nil {
		return err
	}
	// Initialize `response`
	resp := setupResponse(L, w)

	// Setup the `router` module
	L.PreloadModule("router", setupRouter(resp, r.Method, r.URL.Path))
	// TODO(tsileo):
	// - json
	// - http

	// Execute the Lua code
	if err := L.DoString(code); err != nil {
		return err
	}

	// Write `response` content to the HTTP response
	resp.apply()

	return nil
}
