package gluapp

import (
	"reflect"
	"testing"
)

// TODO(tsileo): add a bench

var testRoute = []struct {
	method, path, path2 string
	data, expectedData  string
	expectedParams      Params
}{
	{"GET", "/hello", "/hello", "hello", "hello", Params{}},
	{"POST", "/hello", "/hello", "hellopost", "hellopost", Params{}},
	{"GET", "/", "/", "index", "index", Params{}},
	{"GET", "/hello/:name", "/hello/thomas", "hellop", "hellok", Params{"name": "thomas"}},
	{"GET", "/hello/ok", "/hello/ok", "hellok", "hellop", Params{"name": "ok"}},
	{"GET", "/another/page/:foo/:bar", "/another/page/lol/nope", "foobar", "foobar", Params{"foo": "lol", "bar": "nope"}},
	{"GET", "not:a named/parameter", "not:anamed/parameter", "nnp", "nnp", Params{}},
}

func TestRouter(t *testing.T) {
	r := New()
	check := func(method, path, name string, pExpected Params) {
		route, params := r.Match(method, path)
		if route != nil && route.(string) != name {
			t.Errorf("got %+v expected \"%s\"", route.(string), name)
		}
		if reflect.DeepEqual(params, pExpected) {
			t.Errorf("got %+v expected %+v", params, pExpected)
		}
	}
	for _, testData := range testRoute {
		r.Add(testData.method, testData.path, testData.data)
	}
	for _, testData := range testRoute {
		check(testData.method, testData.path2, testData.data, testData.expectedParams)
	}
}
