/*

The Router implements a basic HTTP router that does not rely on `net/http` at all.

It supports named parameters (`/hello/:name`), insertion order of routes does matter, the first matching route is
returned.

Designed to be used as router for luareq.

*/
package gluapp

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

type route struct {
	path   string
	regexp *regexp.Regexp
	data   interface{}
}

// Params represents a route named parameters
type Params map[string]string

func (r *route) match(path string) (bool, Params) {
	if r.regexp != nil {
		matches := r.regexp.FindStringSubmatch(path)
		if matches != nil {
			params := Params{}
			for i, k := range r.regexp.SubexpNames()[1:] {
				params[k] = matches[i]
			}
			return true, params
		}
	}
	if path == r.path {
		return true, nil
	}
	return false, nil
}

// Router represents the router and holds the routes.
type Router struct {
	routes map[string][]*route
}

func New() *Router {
	return &Router{
		routes: map[string][]*route{},
	}
}

// Add adds the path to the router, order of insertions matters as the first matched route is returned.
func (r *Router) Add(method, path string, data interface{}) {
	// TODO(tsileo): make more verification on the path?
	newRoute := &route{
		data: data,
		path: path,
	}
	if strings.Contains(path, ":") {
		newRoute.path = ""
		parts := strings.Split(path, "/")
		var buf bytes.Buffer
		var hasRegexp bool
		for _, part := range parts {
			// Check if the key is a named parameters
			if strings.HasPrefix(part, ":") && strings.Contains(part, ":") {
				hasRegexp = true
				buf.WriteString(fmt.Sprintf("(?P<%s>[^/]+)", part[1:]))
			} else {
				buf.WriteString(part)
			}
			buf.WriteString("/")
		}
		// Ensure a regex is needed
		if hasRegexp {
			reg := regexp.MustCompile(buf.String())
			newRoute.regexp = reg

		} else {
			// Fallback to basic string matching
			newRoute.path = path
		}
	}
	r.routes[method] = append(r.routes[method], newRoute)
}

// Match returns the given route data alog with the params if any matches
func (r *Router) Match(method, path string) (interface{}, Params) {
	if routes, ok := r.routes[method]; ok {
		for _, rt := range routes {
			match, params := rt.match(path)
			if match {
				return rt.data, params
			}
		}
	}
	return nil, nil
}
