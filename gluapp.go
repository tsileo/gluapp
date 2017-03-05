package gluapp // import "a4.io/gluapp"

import (
	"net/http"

	"github.com/yuin/gopher-lua"
)

// TODO(tsileo): a "app" wrapper that make tu public/tempalte dir/config path optional
// TODO(tsileo): render go template via the path
// TODO(tsileo): render public dir via whitelist, then execute the Lua app
// TODO(tsileo): a logFunc(t time.Time, msg string, args ...interface{})?
// TODO(tsileo): `body, resp, err = http:get('http://...')`
// `body:json()`, resp.statuscode

var methods = []string{
	"GET", "POST", "PUT", "PATCH", "DELETE", "TRACE", "CONNECT", "OPTIONS", "HEAD",
}

// Config represents an app configuration
type Config struct {
	Path string
}

// Exec run the code as a Lua script
func Exec(conf *Config, code string, w http.ResponseWriter, r *http.Request) error {
	// TODO(tsileo): clean error, take L as argument
	L := lua.NewState()
	defer L.Close()

	// Update the path if needed
	if conf.Path != "" {
		path := L.GetField(L.GetField(L.Get(lua.EnvironIndex), "package"), "path").(lua.LString)
		// TODO(tsileo): handle ending path in config.Path
		path = lua.LString(conf.Path + "/?.lua;" + string(path))
		L.SetField(L.GetField(L.Get(lua.EnvironIndex), "package"), "path", lua.LString(path))
	}

	// Setup `request`
	if err := setupRequest(L, r); err != nil {
		return err
	}
	// Initialize `response`
	resp := setupResponse(L, w)

	// Setup the `router` module
	L.PreloadModule("router", setupRouter(resp, r.Method, r.URL.Path))
	L.PreloadModule("json", loadJSON)
	L.PreloadModule("http", loadHTTP)
	// TODO(tsileo): a read/write file module

	// Execute the Lua code
	if err := L.DoString(code); err != nil {
		return err
	}

	// Write `response` content to the HTTP response
	resp.apply()

	return nil
}
