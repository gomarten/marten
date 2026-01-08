package marten

import "net/http"

// Group represents a route group with shared prefix and middleware.
type Group struct {
	prefix     string
	middleware []Middleware
	router     *Router
}

// Group creates a new route group with the given prefix.
func (r *Router) Group(prefix string, mw ...Middleware) *Group {
	return &Group{
		prefix:     prefix,
		middleware: mw,
		router:     r,
	}
}

// Use adds middleware to the group.
func (g *Group) Use(mw ...Middleware) {
	g.middleware = append(g.middleware, mw...)
}

// Group creates a nested group.
func (g *Group) Group(prefix string, mw ...Middleware) *Group {
	return &Group{
		prefix:     g.prefix + prefix,
		middleware: append(g.middleware, mw...),
		router:     g.router,
	}
}

// Handle registers a route within the group.
func (g *Group) Handle(method, path string, h Handler, mw ...Middleware) {
	combined := append(g.middleware, mw...)
	g.router.Handle(method, g.prefix+path, h, combined...)
}

// GET registers a GET route within the group.
func (g *Group) GET(path string, h Handler, mw ...Middleware) {
	g.Handle(http.MethodGet, path, h, mw...)
}

// POST registers a POST route within the group.
func (g *Group) POST(path string, h Handler, mw ...Middleware) {
	g.Handle(http.MethodPost, path, h, mw...)
}

// PUT registers a PUT route within the group.
func (g *Group) PUT(path string, h Handler, mw ...Middleware) {
	g.Handle(http.MethodPut, path, h, mw...)
}

// DELETE registers a DELETE route within the group.
func (g *Group) DELETE(path string, h Handler, mw ...Middleware) {
	g.Handle(http.MethodDelete, path, h, mw...)
}

// PATCH registers a PATCH route within the group.
func (g *Group) PATCH(path string, h Handler, mw ...Middleware) {
	g.Handle(http.MethodPatch, path, h, mw...)
}
