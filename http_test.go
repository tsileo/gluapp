package gluapp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yuin/gopher-lua"
)

func TestHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("http", loadHTTP)

	// Execute the Lua code
	if err := L.DoString(`
print('test HTTP')
http = require('http')
resp, err = http.get('` + server.URL + `')
print(resp.status_code)
print(resp.body:size())
print(resp.body:text())
-- print(resp.body:json())
print(err)

`); err != nil {
		panic(err)
	}
}
