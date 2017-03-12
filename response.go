package gluapp

import (
	"bytes"
	"net/http"

	"github.com/yuin/gopher-lua"
)

// Response represents the HTTP response
type Response struct {
	buf        *bytes.Buffer
	Body       []byte
	Header     http.Header
	StatusCode int
}

// WriteTo dumps the respons to the actual  response.
func (resp *Response) WriteTo(w http.ResponseWriter) {
	resp.Body = resp.buf.Bytes()
	resp.buf = nil

	if w != nil {
		// Write the headers
		for k, vs := range resp.Header {
			// Reset existing values
			w.Header().Del(k)
			if len(vs) == 1 {
				w.Header().Set(k, resp.Header.Get(k))
			}
			if len(vs) > 1 {
				for _, v := range vs {
					w.Header().Add(k, v)
				}
			}
		}

		w.WriteHeader(resp.StatusCode)
		w.Write(resp.Body)
	}
}

func newResponse(L *lua.LState, w http.ResponseWriter) (*lua.LUserData, *Response) {
	resp := &Response{
		buf:        bytes.NewBuffer(nil),
		StatusCode: 200,
		Header:     http.Header{},
	}

	// Copy the headers already set in the response
	for header, vals := range w.Header() {
		for _, v := range vals {
			resp.Header.Add(header, v)
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

func checkResponse(L *lua.LState) *Response {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*Response); ok {
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
	resp.StatusCode = L.ToInt(2)
	return 0
}

func responseWrite(L *lua.LState) int {
	resp := checkResponse(L)
	if resp == nil {
		return 1
	}
	resp.buf.WriteString(L.ToString(2))
	return 0
}

func responseHeaders(L *lua.LState) int {
	resp := checkResponse(L)
	if resp == nil {
		return 1
	}
	L.Push(buildHeaders(L, resp.Header))
	return 1
}

func responseError(L *lua.LState) int {
	resp := checkResponse(L)
	if resp == nil {
		return 1
	}
	status := int(L.ToNumber(2))
	resp.StatusCode = status

	var message string
	if L.GetTop() == 3 {
		message = L.ToString(3)
	} else {
		message = http.StatusText(status)
	}
	resp.buf.Reset()
	resp.buf.WriteString(message)
	return 0
}

func responseAuthenticate(L *lua.LState) int {
	resp := checkResponse(L)
	if resp == nil {
		return 1
	}
	resp.Header.Set("WWW-Authenticate", "Basic realm=\""+L.ToString(2)+"\"")
	return 0
}

func responseJsonify(L *lua.LState) int {
	resp := checkResponse(L)
	if resp == nil {
		return 1
	}
	js := toJSON(L.CheckAny(2))
	resp.buf.Write(js)
	resp.Header.Set("Content-Type", "application/json")
	return 0
}
