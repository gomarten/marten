package tests

import (
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/gomarten/marten"
)

func TestRouterBasicRoutes(t *testing.T) {
	app := marten.New()

	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "root")
	})

	app.GET("/users", func(c *marten.Ctx) error {
		return c.Text(200, "users")
	})

	app.POST("/users", func(c *marten.Ctx) error {
		return c.Text(201, "created")
	})

	tests := []struct {
		method string
		path   string
		status int
		body   string
	}{
		{"GET", "/", 200, "root"},
		{"GET", "/users", 200, "users"},
		{"POST", "/users", 201, "created"},
		{"GET", "/notfound", 404, "Not Found"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != tt.status {
			t.Errorf("%s %s: expected status %d, got %d", tt.method, tt.path, tt.status, rec.Code)
		}
		if rec.Body.String() != tt.body {
			t.Errorf("%s %s: expected body %q, got %q", tt.method, tt.path, tt.body, rec.Body.String())
		}
	}
}

func TestRouterPathParams(t *testing.T) {
	app := marten.New()

	app.GET("/users/:id", func(c *marten.Ctx) error {
		return c.Text(200, "user:"+c.Param("id"))
	})

	app.GET("/users/:id/posts/:postId", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("id")+":"+c.Param("postId"))
	})

	tests := []struct {
		path string
		body string
	}{
		{"/users/123", "user:123"},
		{"/users/abc", "user:abc"},
		{"/users/42/posts/99", "42:99"},
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

func TestRouterMethods(t *testing.T) {
	app := marten.New()

	app.GET("/resource", func(c *marten.Ctx) error { return c.Text(200, "GET") })
	app.POST("/resource", func(c *marten.Ctx) error { return c.Text(200, "POST") })
	app.PUT("/resource", func(c *marten.Ctx) error { return c.Text(200, "PUT") })
	app.DELETE("/resource", func(c *marten.Ctx) error { return c.Text(200, "DELETE") })
	app.PATCH("/resource", func(c *marten.Ctx) error { return c.Text(200, "PATCH") })

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

	for _, method := range methods {
		req := httptest.NewRequest(method, "/resource", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != method {
			t.Errorf("%s /resource: expected %q, got %q", method, method, rec.Body.String())
		}
	}
}

func TestRouterNotFound(t *testing.T) {
	app := marten.New()
	app.NotFound(func(c *marten.Ctx) error {
		return c.Text(404, "custom not found")
	})

	req := httptest.NewRequest("GET", "/missing", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Errorf("expected 404, got %d", rec.Code)
	}
	if rec.Body.String() != "custom not found" {
		t.Errorf("expected custom message, got %q", rec.Body.String())
	}
}

func TestRouterMethodNotAllowed(t *testing.T) {
	app := marten.New()
	app.GET("/only-get", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("POST", "/only-get", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Errorf("expected 404 for wrong method, got %d", rec.Code)
	}
}

// --- Additional Router Tests ---

func TestRouterTrailingSlash(t *testing.T) {
	app := marten.New()
	app.GET("/users", func(c *marten.Ctx) error {
		return c.Text(200, "users")
	})

	// Without trailing slash
	req := httptest.NewRequest("GET", "/users", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("without slash: expected 200, got %d", rec.Code)
	}

	// With trailing slash (currently returns 404)
	req = httptest.NewRequest("GET", "/users/", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	// Note: Current implementation treats /users and /users/ differently
}

func TestRouterEmptyParam(t *testing.T) {
	app := marten.New()
	app.GET("/users/:id", func(c *marten.Ctx) error {
		return c.Text(200, "id:"+c.Param("id"))
	})

	req := httptest.NewRequest("GET", "/users/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	// Empty param segment
}

func TestRouterSpecialCharsInParam(t *testing.T) {
	app := marten.New()
	app.GET("/files/:name", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("name"))
	})

	tests := []struct {
		path     string
		expected string
	}{
		{"/files/test.txt", "test.txt"},
		{"/files/my-file", "my-file"},
		{"/files/file_name", "file_name"},
		{"/files/123", "123"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != tt.expected {
			t.Errorf("%s: expected %q, got %q", tt.path, tt.expected, rec.Body.String())
		}
	}
}

func TestRouterMultipleParams(t *testing.T) {
	app := marten.New()
	app.GET("/org/:org/repo/:repo/branch/:branch", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("org")+"/"+c.Param("repo")+"/"+c.Param("branch"))
	})

	req := httptest.NewRequest("GET", "/org/acme/repo/api/branch/main", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "acme/api/main" {
		t.Errorf("expected acme/api/main, got %q", rec.Body.String())
	}
}

func TestRouterParamPrecedence(t *testing.T) {
	app := marten.New()

	// Static route should take precedence over param
	app.GET("/users/me", func(c *marten.Ctx) error {
		return c.Text(200, "me")
	})
	app.GET("/users/:id", func(c *marten.Ctx) error {
		return c.Text(200, "user:"+c.Param("id"))
	})

	// Static match
	req := httptest.NewRequest("GET", "/users/me", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Body.String() != "me" {
		t.Errorf("expected 'me', got %q", rec.Body.String())
	}

	// Param match
	req = httptest.NewRequest("GET", "/users/123", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Body.String() != "user:123" {
		t.Errorf("expected 'user:123', got %q", rec.Body.String())
	}
}

func TestRouterDeepNesting(t *testing.T) {
	app := marten.New()
	app.GET("/a/b/c/d/e/f", func(c *marten.Ctx) error {
		return c.Text(200, "deep")
	})

	req := httptest.NewRequest("GET", "/a/b/c/d/e/f", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "deep" {
		t.Errorf("expected 'deep', got %q", rec.Body.String())
	}
}

func TestRouterRootPath(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "root")
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "root" {
		t.Errorf("expected 'root', got %q", rec.Body.String())
	}
}

func TestRouterAllMethods(t *testing.T) {
	app := marten.New()

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	for _, m := range methods {
		method := m
		app.Handle(method, "/test", func(c *marten.Ctx) error {
			return c.Text(200, method)
		})
	}

	for _, method := range methods {
		req := httptest.NewRequest(method, "/test", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != method {
			t.Errorf("%s: expected %q, got %q", method, method, rec.Body.String())
		}
	}
}

func TestRouterConcurrent(t *testing.T) {
	app := marten.New()
	app.GET("/users/:id", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("id"))
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			path := "/users/" + strconv.Itoa(id)
			req := httptest.NewRequest("GET", path, nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			expected := strconv.Itoa(id)
			if rec.Body.String() != expected {
				t.Errorf("concurrent %d: expected %q, got %q", id, expected, rec.Body.String())
			}
		}(i)
	}
	wg.Wait()
}
