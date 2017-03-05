package gluapp

import (
	"io/ioutil"
	"net/http"

	"github.com/yuin/gopher-lua"
)

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
		uploadMaxMemory: 32 * 1024 * 1024, // FIXME(tsileo): move this to config
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
