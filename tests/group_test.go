package tests

import (
	"net/http/httptest"
	"testing"

	"github.com/gomarten/marten"
)

func TestRouteGroup(t *testing.T) {
	app := marten.New()

	api := app.Group("/api")
	api.GET("/users", func(c *marten.Ctx) error {
		return c.Text(200, "api users")
	})
	api.GET("/users/:id", func(c *marten.Ctx) error {
		return c.Text(200, "api user:"+c.Param("id"))
	})

	tests := []struct {
		path string
		body string
	}{
		{"/api/users", "api users"},
		{"/api/users/123", "api user:123"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != tt.body {
			t.Errorf("%s: expected %q, got %q", tt.path, tt.body, rec.Body.String())
		}
	}
}

func TestNestedGroups(t *testing.T) {
	app := marten.New()

	api := app.Group("/api")
	v1 := api.Group("/v1")
	users := v1.Group("/users")

	users.GET("", func(c *marten.Ctx) error {
		return c.Text(200, "v1 users list")
	})
	users.GET("/:id", func(c *marten.Ctx) error {
		return c.Text(200, "v1 user:"+c.Param("id"))
	})

	tests := []struct {
		path string
		body string
	}{
		{"/api/v1/users", "v1 users list"},
		{"/api/v1/users/42", "v1 user:42"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != tt.body {
			t.Errorf("%s: expected %q, got %q", tt.path, tt.body, rec.Body.String())
		}
	}
}

func TestGroupMiddleware(t *testing.T) {
	app := marten.New()

	var called bool
	authMw := func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			called = true
			if c.Request.Header.Get("Authorization") == "" {
				return c.Text(401, "unauthorized")
			}
			return next(c)
		}
	}

	// Public routes
	app.GET("/public", func(c *marten.Ctx) error {
		return c.Text(200, "public")
	})

	// Protected group
	admin := app.Group("/admin", authMw)
	admin.GET("/dashboard", func(c *marten.Ctx) error {
		return c.Text(200, "dashboard")
	})

	// Test public route (middleware not called)
	called = false
	req := httptest.NewRequest("GET", "/public", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if called {
		t.Error("auth middleware should not be called for public route")
	}
	if rec.Code != 200 {
		t.Errorf("public: expected 200, got %d", rec.Code)
	}

	// Test admin route without auth
	called = false
	req = httptest.NewRequest("GET", "/admin/dashboard", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if !called {
		t.Error("auth middleware should be called for admin route")
	}
	if rec.Code != 401 {
		t.Errorf("admin without auth: expected 401, got %d", rec.Code)
	}

	// Test admin route with auth
	req = httptest.NewRequest("GET", "/admin/dashboard", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("admin with auth: expected 200, got %d", rec.Code)
	}
}

func TestGroupMethods(t *testing.T) {
	app := marten.New()

	api := app.Group("/api")
	api.GET("/resource", func(c *marten.Ctx) error { return c.Text(200, "GET") })
	api.POST("/resource", func(c *marten.Ctx) error { return c.Text(200, "POST") })
	api.PUT("/resource", func(c *marten.Ctx) error { return c.Text(200, "PUT") })
	api.DELETE("/resource", func(c *marten.Ctx) error { return c.Text(200, "DELETE") })
	api.PATCH("/resource", func(c *marten.Ctx) error { return c.Text(200, "PATCH") })

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		req := httptest.NewRequest(method, "/api/resource", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != method {
			t.Errorf("%s /api/resource: expected %q, got %q", method, method, rec.Body.String())
		}
	}
}

// --- Additional Group Tests ---

func TestGroupMiddlewareInheritance(t *testing.T) {
	app := marten.New()

	var order []string

	parentMw := func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			order = append(order, "parent")
			return next(c)
		}
	}

	childMw := func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			order = append(order, "child")
			return next(c)
		}
	}

	api := app.Group("/api", parentMw)
	v1 := api.Group("/v1", childMw)
	v1.GET("/test", func(c *marten.Ctx) error {
		order = append(order, "handler")
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	expected := []string{"parent", "child", "handler"}
	if len(order) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("position %d: expected %q, got %q", i, v, order[i])
		}
	}
}

func TestGroupUseAfterCreation(t *testing.T) {
	app := marten.New()

	var called bool
	mw := func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			called = true
			return next(c)
		}
	}

	api := app.Group("/api")
	api.Use(mw) // Add middleware after group creation
	api.GET("/test", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !called {
		t.Error("middleware should have been called")
	}
}

func TestGroupRouteSpecificMiddleware(t *testing.T) {
	app := marten.New()

	var groupMwCalled, routeMwCalled bool

	groupMw := func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			groupMwCalled = true
			return next(c)
		}
	}

	routeMw := func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			routeMwCalled = true
			return next(c)
		}
	}

	api := app.Group("/api", groupMw)
	api.GET("/test", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	}, routeMw)

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !groupMwCalled {
		t.Error("group middleware should have been called")
	}
	if !routeMwCalled {
		t.Error("route middleware should have been called")
	}
}

func TestGroupHandle(t *testing.T) {
	app := marten.New()

	api := app.Group("/api")
	api.Handle("GET", "/custom", func(c *marten.Ctx) error {
		return c.Text(200, "custom")
	})

	req := httptest.NewRequest("GET", "/api/custom", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "custom" {
		t.Errorf("expected 'custom', got %q", rec.Body.String())
	}
}

func TestGroupEmptyPrefix(t *testing.T) {
	app := marten.New()

	// Group with empty prefix
	g := app.Group("")
	g.GET("/test", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "ok" {
		t.Errorf("expected 'ok', got %q", rec.Body.String())
	}
}

func TestGroupDeeplyNested(t *testing.T) {
	app := marten.New()

	a := app.Group("/a")
	b := a.Group("/b")
	c := b.Group("/c")
	d := c.Group("/d")
	d.GET("/e", func(c *marten.Ctx) error {
		return c.Text(200, "deep")
	})

	req := httptest.NewRequest("GET", "/a/b/c/d/e", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "deep" {
		t.Errorf("expected 'deep', got %q", rec.Body.String())
	}
}

func TestGroupWithParams(t *testing.T) {
	app := marten.New()

	orgs := app.Group("/orgs/:orgId")
	repos := orgs.Group("/repos/:repoId")
	repos.GET("/info", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("orgId")+":"+c.Param("repoId"))
	})

	req := httptest.NewRequest("GET", "/orgs/acme/repos/api/info", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "acme:api" {
		t.Errorf("expected 'acme:api', got %q", rec.Body.String())
	}
}
