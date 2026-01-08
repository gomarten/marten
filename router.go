package marten

import (
	"net/http"
	"strings"
)

// node represents a node in the radix tree.
type node struct {
	path     string
	children []*node
	param    *node
	wildcard *node
	handlers map[string]Handler
	mw       []Middleware
}

// Router handles HTTP routing with a radix tree.
type Router struct {
	root       *node
	middleware []Middleware
	notFound   Handler
}

// NewRouter creates a new router.
func NewRouter() *Router {
	return &Router{
		root: &node{
			handlers: make(map[string]Handler),
		},
		notFound: func(c *Ctx) error {
			return c.Text(http.StatusNotFound, "Not Found")
		},
	}
}

// Use adds global middleware.
func (r *Router) Use(mw ...Middleware) {
	r.middleware = append(r.middleware, mw...)
}

// NotFound sets a custom 404 handler.
func (r *Router) NotFound(h Handler) {
	r.notFound = h
}

// Handle registers a route with optional route-specific middleware.
func (r *Router) Handle(method, path string, h Handler, mw ...Middleware) {
	parts := splitPath(path)
	current := r.root

	for _, part := range parts {
		current = current.findOrCreate(part)
	}

	if current.handlers == nil {
		current.handlers = make(map[string]Handler)
	}
	current.handlers[method] = h
	current.mw = mw
}

// GET registers a GET route.
func (r *Router) GET(path string, h Handler, mw ...Middleware) {
	r.Handle(http.MethodGet, path, h, mw...)
}

// POST registers a POST route.
func (r *Router) POST(path string, h Handler, mw ...Middleware) {
	r.Handle(http.MethodPost, path, h, mw...)
}

// PUT registers a PUT route.
func (r *Router) PUT(path string, h Handler, mw ...Middleware) {
	r.Handle(http.MethodPut, path, h, mw...)
}

// DELETE registers a DELETE route.
func (r *Router) DELETE(path string, h Handler, mw ...Middleware) {
	r.Handle(http.MethodDelete, path, h, mw...)
}

// PATCH registers a PATCH route.
func (r *Router) PATCH(path string, h Handler, mw ...Middleware) {
	r.Handle(http.MethodPatch, path, h, mw...)
}

func (n *node) findOrCreate(segment string) *node {
	if strings.HasPrefix(segment, "*") {
		if n.wildcard == nil {
			n.wildcard = &node{path: segment}
		}
		return n.wildcard
	}

	if strings.HasPrefix(segment, ":") {
		if n.param == nil {
			n.param = &node{path: segment}
		}
		return n.param
	}

	for _, child := range n.children {
		if child.path == segment {
			return child
		}
	}

	child := &node{path: segment}
	n.children = append(n.children, child)
	return child
}

func (r *Router) lookup(method string, path string, params map[string]string) (Handler, []Middleware) {
	parts := splitPath(path)
	current := r.root

	for i, part := range parts {
		found := false

		for _, child := range current.children {
			if child.path == part {
				current = child
				found = true
				break
			}
		}

		if !found && current.param != nil {
			paramName := current.param.path[1:]
			params[paramName] = part
			current = current.param
			found = true
		}

		if !found && current.wildcard != nil {
			wildcardName := current.wildcard.path[1:]
			remaining := strings.Join(parts[i:], "/")
			params[wildcardName] = remaining
			current = current.wildcard
			break
		}

		if !found {
			return nil, nil
		}
	}

	if h, ok := current.handlers[method]; ok {
		return h, current.mw
	}
	return nil, nil
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}
