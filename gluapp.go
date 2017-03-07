package gluapp // import "a4.io/gluapp"

import (
	"net/http"

	"github.com/yuin/gopher-lua"
)

// TODO(tsileo): a "app" wrapper that make tu public/tempalte dir/config path optional
// TODO(tsileo): render go template via the path
// TODO(tsileo): render public dir via whitelist, then execute the Lua app
// TODO(tsileo): a logFunc(t time.Time, msg string, args ...interface{})?

var methods = []string{
	"GET", "POST", "PUT", "PATCH", "DELETE", "TRACE", "CONNECT", "OPTIONS", "HEAD",
}

// Config represents an app configuration
type Config struct {
	// Path for looking up resources (Lua files, templates, public assets)
	Path string

	// HTTP client, if not set, `http.DefaultClient` will be used
	Client *http.Client
}

func setupState(L *lua.LState) {

}

// Exec run the code as a Lua script
func Exec(conf *Config, code string, w http.ResponseWriter, r *http.Request) error {
	// TODO(tsileo): clean error, take L as argument
	L := lua.NewState()
	defer L.Close()

	// Update the path if needed
	if conf.Path != "" {
		path := L.GetField(L.GetField(L.Get(lua.EnvironIndex), "package"), "path").(lua.LString)
		path = lua.LString(conf.Path + "/?.lua;" + string(path))
		L.SetField(L.GetField(L.Get(lua.EnvironIndex), "package"), "path", lua.LString(path))
	}

	// Setup `request`
	req, err := newRequest(L, r)
	if err != nil {
		return err
	}
	// Initialize `response`
	resp, lresp := newResponse(L, w)

	rootTable := L.CreateTable(0, 2)
	rootTable.RawSetH(lua.LString("request"), req)
	rootTable.RawSetH(lua.LString("response"), resp)
	L.SetGlobal("gluapp", rootTable)

	// Setup the `router` module
	L.PreloadModule("router", setupRouter(lresp, r.Method, r.URL.Path))
	L.PreloadModule("json", loadJSON)

	client := conf.Client
	if client == nil {
		client = http.DefaultClient
	}
	L.PreloadModule("http", setupHTTP(client))

	L.PreloadModule("form", setupForm()) // must be executed after setupHTTP
	L.PreloadModule("template", setupTemplate())
	// TODO(tsileo): a read/write file module???

	// Execute the Lua code
	if err := L.DoString(code); err != nil {
		return err
	}

	// Write `response` content to the HTTP response
	lresp.apply()

	return nil
}
