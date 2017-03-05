package gluapp

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExec(t *testing.T) {
	h := func(w http.ResponseWriter, r *http.Request) {
		Exec(`response:write('ok')`, w, r)
	}
	ts := httptest.NewServer(http.HandlerFunc(h))
	defer ts.Close()

	t.Logf("url=%+v\n", ts.URL)
	res, err := http.Get(ts.URL)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		panic(err)
	}
	t.Logf("body=%s\n", body)
}
