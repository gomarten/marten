package marten

import (
	"fmt"
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
	root          *node
	middleware    []Middleware
	notFound      Handler
	trailingSlash TrailingSlashMode
}

// TrailingSlashMode defines how trailing slashes are handled.
type TrailingSlashMode int

const (
	// TrailingSlashIgnore treats /users and /users/ as the same (default)
	TrailingSlashIgnore TrailingSlashMode = iota
	// TrailingSlashRedirect redirects to the canonical path (301)
	TrailingSlashRedirect
	// TrailingSlashStrict treats /users and /users/ as different routes
	TrailingSlashStrict
)

// NewRouter creates a new router.
func NewRouter() *Router {
	return &Router{
		root: &node{
			handlers: make(map[string]Handler),
		},
		notFound: func(c *Ctx) error {
			_ = c.Text(http.StatusNotFound, "Not Found")
			return nil
		},
		trailingSlash: TrailingSlashIgnore,
	}
}

// SetTrailingSlash configures trailing slash handling.
func (r *Router) SetTrailingSlash(mode TrailingSlashMode) {
	r.trailingSlash = mode
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
// Panics if a conflicting param route is detected (e.g., :id vs :name at same position).
func (r *Router) Handle(method, path string, h Handler, mw ...Middleware) {
	parts := splitPath(path)
	current := r.root

	for _, part := range parts {
		current = current.findOrCreateWithConflictCheck(part, path)
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

// HEAD registers a HEAD route.
func (r *Router) HEAD(path string, h Handler, mw ...Middleware) {
	r.Handle(http.MethodHead, path, h, mw...)
}

// OPTIONS registers an OPTIONS route.
func (r *Router) OPTIONS(path string, h Handler, mw ...Middleware) {
	r.Handle(http.MethodOptions, path, h, mw...)
}

// Routes returns all registered routes for debugging.
func (r *Router) Routes() []Route {
	var routes []Route
	r.collectRoutes(r.root, "", &routes)
	return routes
}

// Route represents a registered route.
type Route struct {
	Method string
	Path   string
}

func (r *Router) collectRoutes(n *node, path string, routes *[]Route) {
	currentPath := path
	if n.path != "" {
		currentPath = path + "/" + n.path
	}

	for method := range n.handlers {
		*routes = append(*routes, Route{Method: method, Path: currentPath})
	}

	for _, child := range n.children {
		r.collectRoutes(child, currentPath, routes)
	}
	if n.param != nil {
		r.collectRoutes(n.param, currentPath, routes)
	}
	if n.wildcard != nil {
		r.collectRoutes(n.wildcard, currentPath, routes)
	}
}

func (n *node) findOrCreateWithConflictCheck(segment, fullPath string) *node {
	if strings.HasPrefix(segment, "*") {
		if n.wildcard == nil {
			n.wildcard = &node{path: segment}
		}
		return n.wildcard
	}

	if strings.HasPrefix(segment, ":") {
		if n.param == nil {
			n.param = &node{path: segment}
		} else if n.param.path != segment {
			// Conflict: different param names at same position
			panic(fmt.Sprintf("route conflict: param '%s' conflicts with existing param '%s' in path '%s'",
				segment, n.param.path, fullPath))
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

func (r *Router) lookup(method string, path string, params map[string]string) (Handler, []Middleware, []string) {
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
			return nil, nil, nil
		}
	}

	// Check if we have a handler at current node
	if h, ok := current.handlers[method]; ok {
		return h, current.mw, nil
	}

	// If no handler but we have a wildcard child, try matching with empty wildcard
	if current.wildcard != nil {
		wildcardName := current.wildcard.path[1:]
		params[wildcardName] = ""
		if h, ok := current.wildcard.handlers[method]; ok {
			return h, current.wildcard.mw, nil
		}
		// Check for allowed methods on wildcard
		if len(current.wildcard.handlers) > 0 {
			allowed := make([]string, 0, len(current.wildcard.handlers))
			for m := range current.wildcard.handlers {
				allowed = append(allowed, m)
			}
			return nil, nil, allowed
		}
	}

	// Path matched but method didn't - collect allowed methods
	if len(current.handlers) > 0 {
		allowed := make([]string, 0, len(current.handlers))
		for m := range current.handlers {
			allowed = append(allowed, m)
		}
		return nil, nil, allowed
	}

	return nil, nil, nil
}

// lookupWithTrailingSlash tries to find a route, and if not found,
// tries the alternate path (with or without trailing slash).
// Returns: handler, middleware, allowed methods, redirect path (if should redirect)
func (r *Router) lookupWithTrailingSlash(method string, path string, params map[string]string) (Handler, []Middleware, []string, string) {
	hasTrailingSlash := len(path) > 1 && strings.HasSuffix(path, "/")

	// In strict mode, trailing slash matters
	if r.trailingSlash == TrailingSlashStrict && hasTrailingSlash {
		// Path has trailing slash - only match if route was registered with trailing slash
		// Since splitPath normalizes, we can't distinguish, so treat as not found
		return nil, nil, nil, ""
	}

	// Lookup with normalized path
	h, mw, allowed := r.lookup(method, path, params)

	if h != nil {
		// Found handler - check if we need to redirect
		if r.trailingSlash == TrailingSlashRedirect && hasTrailingSlash {
			normalizedPath := strings.TrimSuffix(path, "/")
			return nil, nil, nil, normalizedPath
		}
		return h, mw, allowed, ""
	}

	// If we have allowed methods, path exists
	if len(allowed) > 0 {
		if r.trailingSlash == TrailingSlashRedirect && hasTrailingSlash {
			normalizedPath := strings.TrimSuffix(path, "/")
			return nil, nil, nil, normalizedPath
		}
		return nil, nil, allowed, ""
	}

	return nil, nil, nil, ""
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}
