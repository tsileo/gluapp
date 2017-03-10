package gluapp

import (
	"bytes"
	"net/http"

	"github.com/yuin/gopher-lua"
)

// response represents the HTTP response
type response struct {
	body       *bytes.Buffer
	headers    http.Header
	statusCode int
	w          http.ResponseWriter
}

func (resp *response) apply() {
	// Write the headers
	for k, vs := range resp.headers {
		// Reset existing values
		resp.w.Header().Del(k)
		if len(vs) == 1 {
			resp.w.Header().Set(k, resp.headers.Get(k))
		}
		if len(vs) > 1 {
			for _, v := range vs {
				resp.w.Header().Add(k, v)
			}
		}
	}

	resp.w.WriteHeader(resp.statusCode)
	resp.w.Write(resp.body.Bytes())
}

func newResponse(L *lua.LState, w http.ResponseWriter) (*lua.LUserData, *response) {
	resp := &response{
		body:       bytes.NewBuffer(nil),
		statusCode: 200,
		headers:    http.Header{},
		w:          w,
	}

	// Copy the headers already set in the response
	for header, vals := range w.Header() {
		for _, v := range vals {
			resp.headers.Add(header, v)
		}
	}

	// FIXME(tsileo): set the metatable only once?
	mt := L.NewTypeMetatable("response")
	// methods
	responseMethods := map[string]lua.LGFunction{
		"set_status": responseStatus,
		"headers":    responseHeaders,
		"write":      responseWrite,
		"jsonify":    responseJsonify,
		"error":      responseError,
		// TODO(tsileo): see how to deal with basic auth
		// "authenticate": responseAuthenticate,
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

func responseHeaders(L *lua.LState) int {
	resp := checkResponse(L)
	if resp == nil {
		return 1
	}
	L.Push(buildHeaders(L, resp.headers))
	return 1
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
	resp.headers.Set("WWW-Authenticate", "Basic realm=\""+L.ToString(2)+"\"")
	return 0
}

func responseJsonify(L *lua.LState) int {
	resp := checkResponse(L)
	if resp == nil {
		return 1
	}
	js := toJSON(L.CheckAny(2))
	resp.body.Write(js)
	resp.headers.Set("Content-Type", "application/json")
	return 0
}
