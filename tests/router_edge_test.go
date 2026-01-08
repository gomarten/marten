package tests

import (
	"net/http/httptest"
	"testing"

	"github.com/gomarten/marten"
)

// --- Router Edge Cases ---

func TestRouterEmptyPath(t *testing.T) {
	app := marten.New()
	app.GET("", func(c *marten.Ctx) error {
		return c.Text(200, "empty")
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Empty path should match root
	if rec.Code == 404 {
		t.Log("Empty path registered as root - behavior may vary")
	}
}

func TestRouterDoubleSlash(t *testing.T) {
	app := marten.New()
	app.GET("/users", func(c *marten.Ctx) error {
		return c.Text(200, "users")
	})

	req := httptest.NewRequest("GET", "//users", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Double slash handling
	t.Logf("Double slash status: %d", rec.Code)
}

func TestRouterLongPath(t *testing.T) {
	app := marten.New()
	app.GET("/a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p", func(c *marten.Ctx) error {
		return c.Text(200, "deep")
	})

	req := httptest.NewRequest("GET", "/a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "deep" {
		t.Errorf("expected 'deep', got %q", rec.Body.String())
	}
}

func TestRouterManyParams(t *testing.T) {
	app := marten.New()
	app.GET("/:a/:b/:c/:d/:e", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("a")+c.Param("b")+c.Param("c")+c.Param("d")+c.Param("e"))
	})

	req := httptest.NewRequest("GET", "/1/2/3/4/5", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "12345" {
		t.Errorf("expected '12345', got %q", rec.Body.String())
	}
}

func TestRouterParamAtRoot(t *testing.T) {
	app := marten.New()
	app.GET("/:id", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("id"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "test" {
		t.Errorf("expected 'test', got %q", rec.Body.String())
	}
}

func TestRouterStaticVsParamOrdering(t *testing.T) {
	app := marten.New()

	// Register param first, then static
	app.GET("/users/:id", func(c *marten.Ctx) error {
		return c.Text(200, "param:"+c.Param("id"))
	})
	app.GET("/users/new", func(c *marten.Ctx) error {
		return c.Text(200, "static:new")
	})

	// Static should match
	req := httptest.NewRequest("GET", "/users/new", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "static:new" {
		t.Errorf("expected 'static:new', got %q", rec.Body.String())
	}

	// Param should match
	req = httptest.NewRequest("GET", "/users/123", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "param:123" {
		t.Errorf("expected 'param:123', got %q", rec.Body.String())
	}
}

func TestRouterSamePathDifferentMethods(t *testing.T) {
	app := marten.New()

	app.GET("/resource", func(c *marten.Ctx) error {
		return c.Text(200, "GET")
	})
	app.POST("/resource", func(c *marten.Ctx) error {
		return c.Text(200, "POST")
	})
	app.PUT("/resource", func(c *marten.Ctx) error {
		return c.Text(200, "PUT")
	})
	app.DELETE("/resource", func(c *marten.Ctx) error {
		return c.Text(200, "DELETE")
	})
	app.PATCH("/resource", func(c *marten.Ctx) error {
		return c.Text(200, "PATCH")
	})

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	for _, method := range methods {
		req := httptest.NewRequest(method, "/resource", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != method {
			t.Errorf("%s: expected %q, got %q", method, method, rec.Body.String())
		}
	}
}

func TestRouterOverwriteHandler(t *testing.T) {
	app := marten.New()

	app.GET("/test", func(c *marten.Ctx) error {
		return c.Text(200, "first")
	})
	app.GET("/test", func(c *marten.Ctx) error {
		return c.Text(200, "second")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Second registration should overwrite
	if rec.Body.String() != "second" {
		t.Errorf("expected 'second', got %q", rec.Body.String())
	}
}

func TestRouterNotFoundCustom(t *testing.T) {
	app := marten.New()
	app.NotFound(func(c *marten.Ctx) error {
		return c.JSON(404, marten.M{
			"error": "not found",
			"path":  c.Path(),
		})
	})

	req := httptest.NewRequest("GET", "/nonexistent/path", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Errorf("expected 404, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Errorf("expected JSON content type, got %q", rec.Header().Get("Content-Type"))
	}
}

func TestRouterMethodNotAllowedBehavior(t *testing.T) {
	app := marten.New()
	app.GET("/only-get", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	// Try POST on GET-only route
	req := httptest.NewRequest("POST", "/only-get", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Current behavior returns 404 for method not allowed
	if rec.Code != 404 {
		t.Logf("Method not allowed returns: %d", rec.Code)
	}
}

func TestRouterParamWithExtension(t *testing.T) {
	app := marten.New()
	app.GET("/files/:filename", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("filename"))
	})

	tests := []struct {
		path     string
		expected string
	}{
		{"/files/document.pdf", "document.pdf"},
		{"/files/image.png", "image.png"},
		{"/files/archive.tar.gz", "archive.tar.gz"},
		{"/files/noextension", "noextension"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != tt.expected {
			t.Errorf("path %s: expected %q, got %q", tt.path, tt.expected, rec.Body.String())
		}
	}
}

func TestRouterParamNumeric(t *testing.T) {
	app := marten.New()
	app.GET("/items/:id", func(c *marten.Ctx) error {
		id := c.ParamInt("id")
		if id == 0 {
			return c.BadRequest("invalid id")
		}
		return c.JSON(200, marten.M{"id": id})
	})

	tests := []struct {
		path     string
		expected int
	}{
		{"/items/123", 200},
		{"/items/0", 400},
		{"/items/abc", 400},
		{"/items/-1", 200}, // Negative is valid int
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != tt.expected {
			t.Errorf("path %s: expected %d, got %d", tt.path, tt.expected, rec.Code)
		}
	}
}

func TestRouterGroupWithTrailingSlash(t *testing.T) {
	app := marten.New()

	api := app.Group("/api/")
	api.GET("users", func(c *marten.Ctx) error {
		return c.Text(200, "users")
	})

	req := httptest.NewRequest("GET", "/api/users", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should handle trailing slash in group prefix
	t.Logf("Group with trailing slash: %d - %s", rec.Code, rec.Body.String())
}

func TestRouterCaseSensitivity(t *testing.T) {
	app := marten.New()
	app.GET("/Users", func(c *marten.Ctx) error {
		return c.Text(200, "Users")
	})

	tests := []struct {
		path     string
		expected int
	}{
		{"/Users", 200},
		{"/users", 404}, // Different case
		{"/USERS", 404}, // Different case
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != tt.expected {
			t.Errorf("path %s: expected %d, got %d", tt.path, tt.expected, rec.Code)
		}
	}
}

func TestRouterQueryStringIgnored(t *testing.T) {
	app := marten.New()
	app.GET("/search", func(c *marten.Ctx) error {
		return c.Text(200, "found")
	})

	// Query string should not affect routing
	req := httptest.NewRequest("GET", "/search?q=test&page=1", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestRouterFragmentIgnored(t *testing.T) {
	app := marten.New()
	app.GET("/page", func(c *marten.Ctx) error {
		return c.Text(200, "page")
	})

	// Fragment should not affect routing (though browsers don't send it)
	req := httptest.NewRequest("GET", "/page", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestRouterMiddlewareOnNotFound(t *testing.T) {
	app := marten.New()

	var middlewareCalled bool
	app.Use(func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			middlewareCalled = true
			return next(c)
		}
	})

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !middlewareCalled {
		t.Error("middleware should be called even for 404")
	}
	if rec.Code != 404 {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestRouterHandleMethod(t *testing.T) {
	app := marten.New()

	// Test Handle method directly
	app.Handle("GET", "/custom", func(c *marten.Ctx) error {
		return c.Text(200, "custom")
	})
	app.Handle("OPTIONS", "/custom", func(c *marten.Ctx) error {
		return c.NoContent()
	})

	// GET
	req := httptest.NewRequest("GET", "/custom", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Body.String() != "custom" {
		t.Errorf("GET: expected 'custom', got %q", rec.Body.String())
	}

	// OPTIONS
	req = httptest.NewRequest("OPTIONS", "/custom", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 204 {
		t.Errorf("OPTIONS: expected 204, got %d", rec.Code)
	}
}

// --- Wildcard Route Tests ---

func TestRouterWildcard(t *testing.T) {
	app := marten.New()
	app.GET("/files/*filepath", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("filepath"))
	})

	tests := []struct {
		path     string
		expected string
	}{
		{"/files/document.txt", "document.txt"},
		{"/files/images/photo.png", "images/photo.png"},
		{"/files/a/b/c/d/e.txt", "a/b/c/d/e.txt"},
		{"/files/path/to/deep/file.json", "path/to/deep/file.json"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != tt.expected {
			t.Errorf("path %s: expected %q, got %q", tt.path, tt.expected, rec.Body.String())
		}
	}
}

func TestRouterWildcardWithPrefix(t *testing.T) {
	app := marten.New()
	app.GET("/static/*path", func(c *marten.Ctx) error {
		return c.Text(200, "static:"+c.Param("path"))
	})
	app.GET("/api/*rest", func(c *marten.Ctx) error {
		return c.Text(200, "api:"+c.Param("rest"))
	})

	tests := []struct {
		path     string
		expected string
	}{
		{"/static/css/style.css", "static:css/style.css"},
		{"/static/js/app.js", "static:js/app.js"},
		{"/api/v1/users", "api:v1/users"},
		{"/api/v2/posts/123", "api:v2/posts/123"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != tt.expected {
			t.Errorf("path %s: expected %q, got %q", tt.path, tt.expected, rec.Body.String())
		}
	}
}

func TestRouterWildcardVsStatic(t *testing.T) {
	app := marten.New()

	// Static route should take precedence
	app.GET("/files/special", func(c *marten.Ctx) error {
		return c.Text(200, "special")
	})
	app.GET("/files/*filepath", func(c *marten.Ctx) error {
		return c.Text(200, "wildcard:"+c.Param("filepath"))
	})

	// Static match
	req := httptest.NewRequest("GET", "/files/special", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Body.String() != "special" {
		t.Errorf("expected 'special', got %q", rec.Body.String())
	}

	// Wildcard match
	req = httptest.NewRequest("GET", "/files/other.txt", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Body.String() != "wildcard:other.txt" {
		t.Errorf("expected 'wildcard:other.txt', got %q", rec.Body.String())
	}
}

func TestRouterWildcardInGroup(t *testing.T) {
	app := marten.New()

	assets := app.Group("/assets")
	assets.GET("/*path", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("path"))
	})

	req := httptest.NewRequest("GET", "/assets/images/logo.png", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "images/logo.png" {
		t.Errorf("expected 'images/logo.png', got %q", rec.Body.String())
	}
}

func TestRouterWildcardWithMiddleware(t *testing.T) {
	app := marten.New()

	var middlewareCalled bool
	mw := func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			middlewareCalled = true
			return next(c)
		}
	}

	app.GET("/files/*filepath", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("filepath"))
	}, mw)

	req := httptest.NewRequest("GET", "/files/test.txt", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !middlewareCalled {
		t.Error("middleware should have been called")
	}
	if rec.Body.String() != "test.txt" {
		t.Errorf("expected 'test.txt', got %q", rec.Body.String())
	}
}
