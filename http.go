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

// TODO(tsileo): an helper for buiding form values

func setupHTTP(client *http.Client) func(*lua.LState) int {
	return func(L *lua.LState) int {
		// Setup the Lua meta table for the respBody user-defined type
		mtRespBody := L.NewTypeMetatable("respBody")
		L.SetField(mtRespBody, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
			"text": respBodyText,
			"size": respBodySize,
			"json": respBodyJSON,
		}))

		// Setup the Lua meta table for the headers user-defined type
		mtHeaders := L.NewTypeMetatable("values")
		L.SetField(mtHeaders, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
			"add": headersAdd,
			"set": headersSet,
			"del": headersDel,
			"get": headersGet,
			"raw": headersRaw,
		}))

		// Setup the Lua meta table the http (client) user-defined type
		mtHTTP := L.NewTypeMetatable("http")
		clientMethods := map[string]lua.LGFunction{
			"headers": func(L *lua.LState) int {
				client := checkHTTPClient(L)
				headers := buildValues(L, client.header)
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
				if h, ok := lv.Value.(*headers); ok {
					header.Set("Content-Type", "application/x-www-form-urlencoded")
					body = strings.NewReader(url.Values(h.header).Encode())
				}
			default:
				// TODO(tsileo): return an error
			}
		}

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
		out.RawSetH(lua.LString("headers"), buildValues(L, resp.Header))
		out.RawSetH(lua.LString("proto"), lua.LString(resp.Proto))
		out.RawSetH(lua.LString("body"), buildRespBody(L, rbody))

		L.Push(out)
		L.Push(lua.LNil)
		return 2
	}
}

// respBody is a custom type for holding the response body
type respBody struct {
	body []byte
}

func buildRespBody(L *lua.LState, body []byte) lua.LValue {
	ud := L.NewUserData()
	ud.Value = &respBody{body}
	L.SetMetatable(ud, L.GetTypeMetatable("respBody"))
	return ud
}

func checkRespBody(L *lua.LState) *respBody {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*respBody); ok {
		return v
	}
	L.ArgError(1, "respBody expected")
	return nil
}

func respBodySize(L *lua.LState) int {
	respBody := checkRespBody(L)
	L.Push(lua.LNumber(float64(len(respBody.body))))
	return 1
}

func respBodyJSON(L *lua.LState) int {
	respBody := checkRespBody(L)
	// TODO(tsileo): improve from JSON when the payload is invalid
	L.Push(fromJSON(L, respBody.body))
	return 1
}

func respBodyText(L *lua.LState) int {
	respBody := checkRespBody(L)
	L.Push(lua.LString(string(respBody.body)))
	return 1
}

// respBody is a custom type for holding the response body
// TODO(tsileo): use map[string][]string ?
type headers struct {
	header http.Header
}

func initHeaders(L *lua.LState) {
	mt := L.NewTypeMetatable("values")
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"add": headersAdd,
		"set": headersSet,
		"del": headersDel,
		"get": headersGet,
		"raw": headersRaw,
	}))
}

func setupForm() func(*lua.LState) int {
	return func(L *lua.LState) int {
		// Setup the "http" module
		mod := L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
			"new": func(L *lua.LState) int {
				ud := L.NewUserData()
				ud.Value = &headers{http.Header{}}
				L.SetMetatable(ud, L.GetTypeMetatable("values"))
				L.Push(ud)
				return 1
			},
		})
		L.Push(mod)
		return 1
	}
}

func buildValues(L *lua.LState, header http.Header) lua.LValue {
	ud := L.NewUserData()
	ud.Value = &headers{header}
	L.SetMetatable(ud, L.GetTypeMetatable("values"))
	return ud
}

func checkHeaders(L *lua.LState) *headers {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*headers); ok {
		return v
	}
	L.ArgError(1, "values expected")
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
