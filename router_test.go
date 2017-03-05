package gluapp

import (
	"reflect"
	"testing"
)

// TODO(tsileo): add a bench

var testRoute = []struct {
	method, path, path2 string
	data, expectedData  string
	expectedParams      params
}{
	{"GET", "/hello", "/hello", "hello", "hello", params{}},
	{"POST", "/hello", "/hello", "hellopost", "hellopost", params{}},
	{"GET", "/", "/", "index", "index", params{}},
	{"GET", "/hello/:name", "/hello/thomas", "hellop", "hellop", params{"name": "thomas"}},
	{"GET", "/hello/ok", "/hello/ok", "hellok", "hellop", params{"name": "ok"}},
	{"GET", "/another/page/:foo/:bar", "/another/page/lol/nope", "foobar", "foobar", params{"foo": "lol", "bar": "nope"}},
	{"GET", "not:a named/parameter", "not:anamed/parameter", "nnp", "nnp", params{}},
}

func TestRouter(t *testing.T) {
	r := &router{routes: []*route{}}
	check := func(method, path, name string, pExpected params) {
		route, params := r.match(method, path)
		if route != nil && route.(string) != name {
			t.Errorf("got %+v expected \"%s\"", route.(string), name)
		}
		if (len(params) > 0 || len(pExpected) > 0) && !reflect.DeepEqual(params, pExpected) {
			t.Errorf("got %+v expected %+v", params, pExpected)
		}
	}
	for _, testData := range testRoute {
		r.add(testData.method, testData.path, testData.data)
	}
	for _, testData := range testRoute {
		check(testData.method, testData.path2, testData.expectedData, testData.expectedParams)
	}
}
