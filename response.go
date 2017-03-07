package gluapp

import (
	"bytes"
	"net/http"

	"github.com/yuin/gopher-lua"
)

// response represents the HTTP response
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

func newResponse(L *lua.LState, w http.ResponseWriter) (*lua.LUserData, *response) {
	resp := &response{
		body:       bytes.NewBuffer(nil),
		statusCode: 200,
		headers:    map[string]string{}, // FIXME(tsileo): use http.Header
		w:          w,
	}
	for header, val := range w.Header() {
		// TODO(tsileo): handle []string for header
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
	return ud, resp
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
