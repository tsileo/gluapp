package gluapp

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/yuin/gopher-lua"
)

func setupHTTP(client *http.Client) func(*lua.LState) int {
	return func(L *lua.LState) int {
		// Setup the Lua meta table the http (client) user-defined type
		mtHTTP := L.NewTypeMetatable("http")
		clientMethods := map[string]lua.LGFunction{
			"headers": func(L *lua.LState) int {
				client := checkHTTPClient(L)
				headers := buildHeaders(L, client.header)
				L.Push(headers)
				return 1
			},
			"set_basic_auth": func(L *lua.LState) int {
				client := checkHTTPClient(L)
				client.username = string(L.ToString(2))
				client.password = string(L.ToString(3))
				return 0
			},
		}
		for _, m := range methods {
			clientMethods[strings.ToLower(m)] = httpClientDoReq(m)
		}
		L.SetField(mtHTTP, "__index", L.SetFuncs(L.NewTable(), clientMethods))

		// Setup the "http" module
		mod := L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
			"new": func(L *lua.LState) int {
				router := &httpClient{
					client: client,
					header: http.Header{},
				}
				ud := L.NewUserData()
				ud.Value = router
				L.SetMetatable(ud, L.GetTypeMetatable("http"))
				L.Push(ud)
				return 1
			},
		})
		L.Push(mod)
		return 1
	}
}

type httpClient struct {
	client   *http.Client
	header   http.Header
	username string
	password string
}

func checkHTTPClient(L *lua.LState) *httpClient {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*httpClient); ok {
		return v
	}
	L.ArgError(1, "http expected")
	return nil
}

func httpClientDoReq(method string) func(*lua.LState) int {
	return func(L *lua.LState) int {
		client := checkHTTPClient(L)
		rurl := L.ToString(2)

		header := http.Header{}

		// Set the body if provided
		var body io.Reader
		if L.GetTop() == 3 {
			switch lv := L.Get(3).(type) {
			case lua.LString:
				body = strings.NewReader(string(lv))
			case *lua.LTable:
				header.Set("Content-Type", "application/json")
				body = bytes.NewReader(toJSON(L.Get(3)))
			case *lua.LUserData:
				if h, ok := lv.Value.(*values); ok {
					header.Set("Content-Type", "application/x-www-form-urlencoded")
					body = strings.NewReader(h.values.Encode())
				}
			default:
				// TODO(tsileo): return an error
			}
		}

		// Create the request
		request, err := http.NewRequest(method, rurl, body)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}

		// Set basic auth if needed
		if client.username != "" || client.password != "" {
			request.SetBasicAuth(client.username, client.password)
		}

		// Add the headers set by the client to the request
		for _, hs := range []http.Header{header, client.header} {
			for k, vs := range hs {
				// Reset existing values
				request.Header.Del(k)
				if len(vs) == 1 {
					request.Header.Set(k, hs.Get(k))
				}
				if len(vs) > 1 {
					for _, v := range vs {
						request.Header.Add(k, v)
					}
				}
			}
		}

		resp, err := client.client.Do(request)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		defer resp.Body.Close()

		// Read the request body
		rbody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}

		out := L.CreateTable(0, 5)
		out.RawSetH(lua.LString("status_code"), lua.LNumber(float64(resp.StatusCode)))
		out.RawSetH(lua.LString("status_line"), lua.LString(resp.Status))
		out.RawSetH(lua.LString("headers"), buildHeaders(L, resp.Header))
		out.RawSetH(lua.LString("proto"), lua.LString(resp.Proto))
		out.RawSetH(lua.LString("body"), buildBody(L, rbody))

		L.Push(out)
		L.Push(lua.LNil)
		return 2
	}
}

// body is a custom type for holding requests/responses body
type body struct {
	data []byte
}

func buildBody(L *lua.LState, data []byte) lua.LValue {
	ud := L.NewUserData()
	ud.Value = &body{data}
	L.SetMetatable(ud, L.GetTypeMetatable("body"))
	return ud
}

func checkBody(L *lua.LState) *body {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*body); ok {
		return v
	}
	L.ArgError(1, "respBody expected")
	return nil
}

func bodySize(L *lua.LState) int {
	body := checkBody(L)
	L.Push(lua.LNumber(float64(len(body.data))))
	return 1
}

func bodyJSON(L *lua.LState) int {
	body := checkBody(L)
	// TODO(tsileo): improve from JSON when the payload is invalid
	L.Push(fromJSON(L, body.data))
	return 1
}

func bodyText(L *lua.LState) int {
	body := checkBody(L)
	L.Push(lua.LString(string(body.data)))
	return 1
}

// respBody is a custom type for holding the response body
type headers struct {
	header http.Header
}

// values holds query parameters or form data
type values struct {
	values url.Values
}

func setupForm() func(*lua.LState) int {
	return func(L *lua.LState) int {
		// Setup the "http" module
		mod := L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
			"new": func(L *lua.LState) int {
				ud := L.NewUserData()
				ud.Value = &values{url.Values{}}
				L.SetMetatable(ud, L.GetTypeMetatable("values"))
				L.Push(ud)
				return 1
			},
		})
		L.Push(mod)
		return 1
	}
}

func buildValues(L *lua.LState, vals url.Values) lua.LValue {
	ud := L.NewUserData()
	ud.Value = &values{vals}
	L.SetMetatable(ud, L.GetTypeMetatable("values"))
	return ud
}

func buildHeaders(L *lua.LState, header http.Header) lua.LValue {
	ud := L.NewUserData()
	ud.Value = &headers{header}
	L.SetMetatable(ud, L.GetTypeMetatable("headers"))
	return ud
}

func checkValues(L *lua.LState) *values {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*values); ok {
		return v
	}
	L.ArgError(1, "values expected")
	return nil
}

func checkHeaders(L *lua.LState) *headers {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*headers); ok {
		return v
	}
	L.ArgError(1, "headers expected")
	return nil
}

func headersAdd(L *lua.LState) int {
	headers := checkHeaders(L)
	headers.header.Add(string(L.ToString(2)), string(L.ToString(3)))
	return 0
}
func headersSet(L *lua.LState) int {
	headers := checkHeaders(L)
	headers.header.Set(string(L.ToString(2)), string(L.ToString(3)))
	return 0
}

func headersDel(L *lua.LState) int {
	headers := checkHeaders(L)
	headers.header.Del(string(L.ToString(2)))
	return 0
}

func headersGet(L *lua.LState) int {
	headers := checkHeaders(L)
	val := headers.header.Get(string(L.ToString(2)))
	L.Push(lua.LString(val))
	return 1
}

func headersRaw(L *lua.LState) int {
	headers := checkHeaders(L)
	out := L.CreateTable(0, len(headers.header))
	for k, vs := range headers.header {
		values := L.CreateTable(len(vs), 0)
		for _, v := range vs {
			values.Append(lua.LString(v))
		}
		out.RawSetH(lua.LString(k), values)
	}
	L.Push(out)
	return 1
}

func valuesAdd(L *lua.LState) int {
	values := checkValues(L)
	values.values.Add(string(L.ToString(2)), string(L.ToString(3)))
	return 0
}

func valuesSet(L *lua.LState) int {
	values := checkValues(L)
	values.values.Set(string(L.ToString(2)), string(L.ToString(3)))
	return 0
}

func valuesDel(L *lua.LState) int {
	values := checkValues(L)
	values.values.Del(string(L.ToString(2)))
	return 0
}

func valuesGet(L *lua.LState) int {
	values := checkValues(L)
	val := values.values.Get(string(L.ToString(2)))
	L.Push(lua.LString(val))
	return 1
}

func valuesRaw(L *lua.LState) int {
	values := checkValues(L)
	out := L.CreateTable(0, len(values.values))
	for k, vs := range values.values {
		values := L.CreateTable(len(vs), 0)
		for _, v := range vs {
			values.Append(lua.LString(v))
		}
		out.RawSetH(lua.LString(k), values)
	}
	L.Push(out)
	return 1
}