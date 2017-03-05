package gluapp

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

var testApp1 = `
response:write('Hello World!')
`

var testAppRouter1 = `
router = require('router').new()
router:get('/hello/:name', function(params)
  response:write('hello ' .. params.name)
  print('hello')
end)
router:run()
`

func TestExec(t *testing.T) {
	h1 := func(w http.ResponseWriter, r *http.Request) {
		if err := Exec(testApp1, w, r); err != nil {
			panic(err)
		}
	}

	h2 := func(w http.ResponseWriter, r *http.Request) {
		if err := Exec(testAppRouter1, w, r); err != nil {
			panic(err)
		}
	}

	servers := map[string]*httptest.Server{
		"s1": httptest.NewServer(http.HandlerFunc(h1)),
		"s2": httptest.NewServer(http.HandlerFunc(h2)),
	}

	for _, server := range servers {
		defer server.Close()
	}

	testData := []struct {
		server                     *httptest.Server
		method                     string
		path                       string
		body                       bytes.Buffer
		expectedResponseBody       string
		expectedResponseStatusCode int
	}{
		{
			method:                     "GET",
			server:                     servers["s1"],
			path:                       "/",
			expectedResponseBody:       "Hello World!",
			expectedResponseStatusCode: 200,
		},
		{
			method:                     "GET",
			server:                     servers["s1"],
			path:                       "/foo",
			expectedResponseBody:       "Hello World!",
			expectedResponseStatusCode: 200,
		},
		{
			method:                     "GET",
			server:                     servers["s2"],
			path:                       "/hello/thomas/",
			expectedResponseBody:       "hello /hello/thomas/", // FIXME(tsileo) fix the slash issue
			expectedResponseStatusCode: 200,
		},
	}

	for _, tdata := range testData {
		var resp *http.Response
		var err error
		switch tdata.method {
		case "GET":
			resp, err = http.Get(tdata.server.URL + tdata.path)
			if err != nil {
				panic(err)
			}
		}
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			panic(err)
		}
		t.Logf("body=%s\n", body)
		if resp.StatusCode != tdata.expectedResponseStatusCode {
			t.Errorf("bad status code, got %d, expected %d", resp.StatusCode, tdata.expectedResponseStatusCode)
		}
		if string(body) != tdata.expectedResponseBody {
			t.Errorf("bad body, got %s, expected %s", body, tdata.expectedResponseBody)
		}
	}
}
