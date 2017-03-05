package gluapp // import "a4.io/gluapp"

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/yuin/gopher-lua"
)

var methods = []string{
	"GET", "POST", "PUT", "PATCH", "DELETE", "TRACE", "CONNECT", "OPTIONS", "HEAD",
}

type request struct {
	uploadMaxMemory int64
	request         *http.Request
	body            []byte // Cache the body, since it can only be streamed once
}

func setupRequest(L *lua.LState, r *http.Request) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	req := &request{
		uploadMaxMemory: 32 * 1024 * 1024,
		request:         r,
		body:            body,
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
	js := toJSON(L.CheckAny(2))
	resp.body.Write(js)
	resp.headers["Content-Type"] = "application/json"
	return 0
}

// Exec run the code as a Lua script
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
	L.PreloadModule("router", setupRouter(resp, r.Method, r.URL.Path))

	// Execute the Lua code
	if err := L.DoString(code); err != nil {
		return err
	}

	// Write `response` content to the HTTP response
	resp.apply()

	return nil
}
