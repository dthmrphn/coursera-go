package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

func match(path, pattern string) (map[string]string, bool) {
	m := map[string]string{}

	for path != "" && pattern != "" {
		switch pattern[0] {
		case '$':
			slash := strings.IndexByte(pattern, '/')
			if slash < 0 {
				slash = len(pattern)
			}

			key := pattern[1:slash]
			pattern = pattern[slash:]

			slash = strings.IndexByte(path, '/')
			if slash < 0 {
				slash = len(path)
			}

			val := path[0:slash]
			path = path[slash:]

			m[key] = val

		case path[0]:
			path = path[1:]
			pattern = pattern[1:]
		}
	}

	return m, path == pattern
}

type route struct {
	handler http.HandlerFunc
	pattern string
}

type contextkey struct{}

type Router struct {
	routes map[string][]route
}

func NewRouter() *Router {
	return &Router{
		routes: map[string][]route{},
	}
}

func (rt *Router) Route(method, pattern string, handler http.HandlerFunc) {
	r := route{handler, pattern}
	rt.routes[method] = append(rt.routes[method], r)
}

func (rt *Router) Serve(w http.ResponseWriter, r *http.Request) {
	routes, ok := rt.routes[r.Method]
	if !ok {
		http.Error(w, "method is unsupported", http.StatusBadRequest)
		return
	}

	for _, route := range routes {
		vals, ok := match(r.URL.Path, route.pattern)
		if ok {
			ctx := context.WithValue(r.Context(), contextkey{}, vals)
			route.handler(w, r.WithContext(ctx))
			return
		}
	}

	http.Error(w, "bad request", http.StatusBadRequest)
}

func URLParam(r *http.Request, key string) string {
	return r.FormValue(key)
}

func RouteParam(r *http.Request, key string) string {
	m := r.Context().Value(contextkey{}).(map[string]string)
	if m == nil {
		return ""
	}

	val, ok := m[key]
	if !ok {
		return ""
	}

	return val
}

func JSONParam(r *http.Request, key string) string {
	d := json.NewDecoder(r.Body)
	m := map[string]string{}
	err := d.Decode(&m)
	if err != nil {
	      return ""
	}

	val, ok := m[key]
	if !ok {
		return ""
	}

	return val
}
