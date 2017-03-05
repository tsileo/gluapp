package gluapp

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"

	"a4.io/blobstash/pkg/apps/luautil"
	"github.com/yuin/gopher-lua"
)

var methods = []string{
	"GET", "POST", "PUT", "PATCH", "DELETE", "TRACE", "CONNECT", "OPTIONS", "HEAD",
}

const any = "any"

type router struct {
	r   *Router
	req *http.Request
}

func setupRouter(req *http.Request) func(*lua.LState) int {
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
				router := &router{r: New(), req: req}
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
				router.r.Add(m, path, fn)
			}

		} else {
			router.r.Add(method, path, fn)
		}
		return 0
	}
}

func routerRun(L *lua.LState) int {
	router := checkRouter(L)
	if router == nil {
		return 1
	}
	// TODO(tsileo) get request struct from req object to do the match
	fn, params := router.r.Match("GET", "/")
	if err := L.CallByParam(lua.P{
		Fn:      lua.LValue(fn.(*lua.LFunction)),
		NRet:    0,
		Protect: true,
	}, luautil.InterfaceToLValue(L, params)); err != nil {
		panic(err)
	}
	return 0
}

type request struct {
	uploadMaxMemory int64
	request         *http.Request
	reqID           string
	cache           []byte // Cache the body, since it can only be streamed once
}

func setupRequest(L *lua.LState, r *http.Request) error {
	cache, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	// TODO(tsileo): reqID still nedeed?
	req := &request{
		uploadMaxMemory: 32 * 1024 * 1024,
		request:         r,
		cache:           cache,
	}
	mt := L.NewTypeMetatable("request")
	// methods
	requestMethods := map[string]lua.LGFunction{}
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), requestMethods))
	ud := L.NewUserData()
	ud.Value = req
	L.SetMetatable(ud, L.GetTypeMetatable("request"))
	L.SetGlobal("request", ud)
	return nil
}

type response struct {
	body       *bytes.Buffer
	headers    map[string]string
	statusCode int
	w          http.ResponseWriter
}

func (resp *response) apply() {
	for header, val := range resp.headers {
		resp.w.Header().Set(header, val)
	}
	resp.w.WriteHeader(resp.statusCode)
	resp.w.Write(resp.body.Bytes())
}

func setupResponse(L *lua.LState, w http.ResponseWriter) *response {
	resp := &response{
		body:       bytes.NewBuffer(nil),
		statusCode: 200,
		headers:    map[string]string{}, // FIXME(tsileo): copy actual resp headers from ResponseWriter
		w:          w,
	}
	for header, val := range w.Header() {
		// FIXME(tsileo): handle []string for header
		resp.headers[header] = val[0]
	}

	mt := L.NewTypeMetatable("response")
	// methods
	responseMethods := map[string]lua.LGFunction{
		"status":       responseStatus,
		"header":       responseHeader, // FIXME(tsileo): add "headers" and return a table with resp
		"write":        responseWrite,
		"error":        responseError,
		"authenticate": responseAuthenticate,
		"jsonify":      responseJsonify,
	}
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), responseMethods))
	ud := L.NewUserData()
	ud.Value = resp
	L.SetMetatable(ud, L.GetTypeMetatable("response"))
	L.SetGlobal("response", ud)
	return resp
}

func checkResponse(L *lua.LState) *response {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*response); ok {
		return v
	}
	L.ArgError(1, "response expected")
	return nil
}

func responseStatus(L *lua.LState) int {
	resp := checkResponse(L)
	if resp == nil {
		return 1
	}
	resp.statusCode = L.ToInt(2)
	return 0
}

func responseWrite(L *lua.LState) int {
	resp := checkResponse(L)
	if resp == nil {
		return 1
	}
	resp.body.WriteString(L.ToString(2))
	return 0
}

func responseHeader(L *lua.LState) int {
	resp := checkResponse(L)
	if resp == nil {
		return 1
	}
	// TODO(tsileo): return the header if no 3rd arg is provided
	resp.headers[L.ToString(2)] = L.ToString(3)
	return 0
}

func responseError(L *lua.LState) int {
	resp := checkResponse(L)
	if resp == nil {
		return 1
	}
	status := int(L.ToNumber(2))
	resp.statusCode = status

	var message string
	if L.GetTop() == 3 {
		message = L.ToString(3)
	} else {
		message = http.StatusText(status)
	}
	resp.body.Reset()
	resp.body.WriteString(message)
	return 0
}

func responseAuthenticate(L *lua.LState) int {
	resp := checkResponse(L)
	if resp == nil {
		return 1
	}
	resp.headers["WWW-Authenticate"] = "Basic realm=\"" + L.ToString(2) + "\""
	return 0
}

func responseJsonify(L *lua.LState) int {
	resp := checkResponse(L)
	if resp == nil {
		return 1
	}
	js := luautil.ToJSON(L.CheckAny(2))
	resp.body.Write(js)
	resp.headers["Content-Type"] = "application/json"
	return 0
}

// var Code = `
// router = require('router').new()
// router:get('/', function(params)
//   print(params)
//   print('index from userfunc')
//   response:jsonify{ok = 1}
//   response:status(500)
// end)
// router:run()
// `

func Exec(code string, w http.ResponseWriter, r *http.Request) error {
	// TODO(tsileo): clean error
	L := lua.NewState()
	defer L.Close()

	// Setup `request`
	if err := setupRequest(L, r); err != nil {
		return err
	}
	// Initialize `response`
	resp := setupResponse(L, w)

	// Setup the `router` module
	L.PreloadModule("router", setupRouter(r))

	// Execute the Lua code
	if err := L.DoString(code); err != nil {
		return err
	}

	// Write `response` content to the HTTP response
	resp.apply()

	return nil
}
