package gluapp

import (
	"io/ioutil"
	"net/http"

	"github.com/yuin/gopher-lua"
)

// request represents the incoming HTTP request
type request struct {
	uploadMaxMemory int64
	request         *http.Request
	body            []byte // Cache the body, since it can only be streamed once
}

func newRequest(L *lua.LState, r *http.Request) (*lua.LUserData, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	req := &request{
		uploadMaxMemory: 32 * 1024 * 1024, // FIXME(tsileo): move this to config
		request:         r,
		body:            body,
	}
	mt := L.NewTypeMetatable("request")
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{}))
	ud := L.NewUserData()
	ud.Value = req
	L.SetMetatable(ud, L.GetTypeMetatable("request"))
	return ud, nil
}
