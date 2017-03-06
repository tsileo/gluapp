package gluapp

import (
	"io/ioutil"
	"net/http"

	"github.com/yuin/gopher-lua"
)

// TODO(tsileo): a helper for basic auth

func loadHTTP(L *lua.LState) int {
	// register functions to the table
	mod := L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"get": httpGet,
	})
	// returns the module
	L.Push(mod)
	return 1
}

func httpGet(L *lua.LState) int {
	url := L.ToString(1)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	// TODO(tsileo): client from Config
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	defer resp.Body.Close()

	// Read the request body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	headers := L.CreateTable(0, len(resp.Header))
	for k, v := range resp.Header {
		headers.RawSetH(lua.LString(k), lua.LString(v[0]))
	}

	out := L.CreateTable(0, 5)
	out.RawSetH(lua.LString("status_code"), lua.LNumber(float64(resp.StatusCode)))
	out.RawSetH(lua.LString("status"), lua.LString(resp.Status))
	out.RawSetH(lua.LString("headers"), headers)
	out.RawSetH(lua.LString("proto"), lua.LString(resp.Proto))
	out.RawSetH(lua.LString("body"), buildRespBody(L, body))

	L.Push(out)
	L.Push(lua.LNil)
	return 2
}

// respBody is a custom type for holding the response body
type respBody struct {
	body []byte
}

func buildRespBody(L *lua.LState, body []byte) lua.LValue {
	mt := L.NewTypeMetatable("respBody")
	// methods
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"text": respBodyText,
		"size": respBodySize,
		"json": respBodyJSON,
	}))
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
