package tests

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gomarten/marten"
	"github.com/gomarten/marten/middleware"
)

// Test HEAD and OPTIONS route methods
func TestRouterHEADAndOPTIONS(t *testing.T) {
	app := marten.New()

	app.HEAD("/resource", func(c *marten.Ctx) error {
		c.Header("X-Custom", "head-value")
		return c.NoContent()
	})

	app.OPTIONS("/resource", func(c *marten.Ctx) error {
		c.Header("Allow", "GET, POST, OPTIONS")
		return c.NoContent()
	})

	// Test HEAD
	req := httptest.NewRequest(http.MethodHead, "/resource", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("HEAD expected 204, got %d", rec.Code)
	}
	if rec.Header().Get("X-Custom") != "head-value" {
		t.Error("HEAD missing custom header")
	}

	// Test OPTIONS
	req = httptest.NewRequest(http.MethodOptions, "/resource", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("OPTIONS expected 204, got %d", rec.Code)
	}
	if rec.Header().Get("Allow") != "GET, POST, OPTIONS" {
		t.Error("OPTIONS missing Allow header")
	}
}

// Test Group HEAD and OPTIONS
func TestGroupHEADAndOPTIONS(t *testing.T) {
	app := marten.New()
	api := app.Group("/api")

	api.HEAD("/ping", func(c *marten.Ctx) error {
		return c.NoContent()
	})

	api.OPTIONS("/ping", func(c *marten.Ctx) error {
		c.Header("Allow", "HEAD, OPTIONS")
		return c.NoContent()
	})

	req := httptest.NewRequest(http.MethodHead, "/api/ping", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rec.Code)
	}
}

// Test Routes() method
func TestRouterRoutes(t *testing.T) {
	app := marten.New()

	app.GET("/users", func(c *marten.Ctx) error { return nil })
	app.POST("/users", func(c *marten.Ctx) error { return nil })
	app.GET("/users/:id", func(c *marten.Ctx) error { return nil })

	routes := app.Routes()

	if len(routes) < 3 {
		t.Errorf("expected at least 3 routes, got %d", len(routes))
	}

	// Check that routes are collected
	found := make(map[string]bool)
	for _, r := range routes {
		found[r.Method+" "+r.Path] = true
	}

	if !found["GET /users"] {
		t.Error("missing GET /users route")
	}
	if !found["POST /users"] {
		t.Error("missing POST /users route")
	}
}

// Test GetHeader method
func TestContextGetHeader(t *testing.T) {
	app := marten.New()

	app.GET("/", func(c *marten.Ctx) error {
		auth := c.GetHeader("Authorization")
		custom := c.GetHeader("X-Custom")
		return c.OK(marten.M{"auth": auth, "custom": custom})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("X-Custom", "custom-value")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "token123") {
		t.Error("GetHeader failed for Authorization")
	}
	if !strings.Contains(body, "custom-value") {
		t.Error("GetHeader failed for X-Custom")
	}
}

// Test Written method
func TestContextWritten(t *testing.T) {
	app := marten.New()

	app.GET("/", func(c *marten.Ctx) error {
		if c.Written() {
			return c.Text(500, "already written")
		}
		_ = c.Text(200, "first")
		if !c.Written() {
			return c.Text(500, "should be written")
		}
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "first" {
		t.Errorf("expected 'first', got '%s'", rec.Body.String())
	}
}

// Test HTML method
func TestContextHTML(t *testing.T) {
	app := marten.New()

	app.GET("/", func(c *marten.Ctx) error {
		return c.HTML(200, "<h1>Hello</h1>")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Error("expected text/html content type")
	}
	if rec.Body.String() != "<h1>Hello</h1>" {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}
}

// Test Blob method
func TestContextBlob(t *testing.T) {
	app := marten.New()

	data := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header

	app.GET("/", func(c *marten.Ctx) error {
		return c.Blob(200, "image/png", data)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "image/png" {
		t.Error("expected image/png content type")
	}
	if !bytes.Equal(rec.Body.Bytes(), data) {
		t.Error("blob data mismatch")
	}
}

// Test Stream method
func TestContextStream(t *testing.T) {
	app := marten.New()

	app.GET("/", func(c *marten.Ctx) error {
		reader := strings.NewReader("streaming data")
		return c.Stream(200, "text/plain", reader)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "streaming data" {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}
}

// Test QueryParams method
func TestContextQueryParams(t *testing.T) {
	app := marten.New()

	app.GET("/", func(c *marten.Ctx) error {
		params := c.QueryParams()
		return c.OK(marten.M{
			"a": params.Get("a"),
			"b": params["b"],
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/?a=1&b=2&b=3", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `"a":"1"`) {
		t.Error("QueryParams failed for 'a'")
	}
}

// Test RateLimit headers
func TestRateLimitHeaders(t *testing.T) {
	rl := middleware.NewRateLimiter(middleware.RateLimitConfig{
		Requests: 5,
		Window:   time.Minute,
	})
	defer rl.Stop()

	app := marten.New()
	app.Use(rl.Middleware())
	app.GET("/", func(c *marten.Ctx) error {
		return c.OK(marten.M{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-RateLimit-Limit") != "5" {
		t.Error("missing X-RateLimit-Limit header")
	}
	if rec.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("missing X-RateLimit-Remaining header")
	}
	if rec.Header().Get("X-RateLimit-Reset") == "" {
		t.Error("missing X-RateLimit-Reset header")
	}
}

// Test RateLimit exceeded
func TestRateLimitExceeded(t *testing.T) {
	rl := middleware.NewRateLimiter(middleware.RateLimitConfig{
		Requests: 2,
		Window:   time.Minute,
	})
	defer rl.Stop()

	app := marten.New()
	app.Use(rl.Middleware())
	app.GET("/", func(c *marten.Ctx) error {
		return c.OK(marten.M{"ok": true})
	})

	// Make 3 requests, 3rd should be rate limited
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if i < 2 && rec.Code != 200 {
			t.Errorf("request %d: expected 200, got %d", i, rec.Code)
		}
		if i == 2 && rec.Code != 429 {
			t.Errorf("request %d: expected 429, got %d", i, rec.Code)
		}
	}
}

// Test RateLimit Skip
func TestRateLimitSkip(t *testing.T) {
	rl := middleware.NewRateLimiter(middleware.RateLimitConfig{
		Requests: 1,
		Window:   time.Minute,
		Skip: func(c *marten.Ctx) bool {
			return c.GetHeader("X-Skip-RateLimit") == "true"
		},
	})
	defer rl.Stop()

	app := marten.New()
	app.Use(rl.Middleware())
	app.GET("/", func(c *marten.Ctx) error {
		return c.OK(marten.M{"ok": true})
	})

	// First request uses the limit
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Second request should be rate limited
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 429 {
		t.Errorf("expected 429, got %d", rec.Code)
	}

	// Third request with skip header should pass
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Skip-RateLimit", "true")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("expected 200 with skip, got %d", rec.Code)
	}
}

// Test Logger with config
func TestLoggerWithConfig(t *testing.T) {
	var buf bytes.Buffer

	app := marten.New()
	app.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Output: &buf,
		Format: func(method, path string, status int, duration time.Duration, clientIP string) string {
			return method + " " + path
		},
	}))
	app.GET("/test", func(c *marten.Ctx) error {
		return c.OK(marten.M{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(buf.String(), "GET /test") {
		t.Errorf("logger output missing expected content: %s", buf.String())
	}
}

// Test Logger Skip
func TestLoggerSkip(t *testing.T) {
	var buf bytes.Buffer

	app := marten.New()
	app.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Output: &buf,
		Skip: func(c *marten.Ctx) bool {
			return c.Path() == "/health"
		},
	}))
	app.GET("/health", func(c *marten.Ctx) error {
		return c.OK(marten.M{"ok": true})
	})
	app.GET("/api", func(c *marten.Ctx) error {
		return c.OK(marten.M{"ok": true})
	})

	// Health should be skipped
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if strings.Contains(buf.String(), "/health") {
		t.Error("logger should skip /health")
	}

	// API should be logged
	req = httptest.NewRequest(http.MethodGet, "/api", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(buf.String(), "/api") {
		t.Error("logger should log /api")
	}
}

// Test Group middleware slice mutation fix
func TestGroupMiddlewareNoMutation(t *testing.T) {
	app := marten.New()

	var order []string

	mw1 := func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			order = append(order, "mw1")
			return next(c)
		}
	}

	mw2 := func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			order = append(order, "mw2")
			return next(c)
		}
	}

	group := app.Group("/api", mw1)

	// Register first route with additional middleware
	group.GET("/a", func(c *marten.Ctx) error {
		order = append(order, "a")
		return c.Text(200, "a")
	}, mw2)

	// Register second route without additional middleware
	group.GET("/b", func(c *marten.Ctx) error {
		order = append(order, "b")
		return c.Text(200, "b")
	})

	// Test /api/a
	order = nil
	req := httptest.NewRequest(http.MethodGet, "/api/a", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Test /api/b - should NOT have mw2
	order = nil
	req = httptest.NewRequest(http.MethodGet, "/api/b", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	for _, o := range order {
		if o == "mw2" {
			t.Error("mw2 should not be applied to /api/b")
		}
	}
}

// Test compress flushes buffered data
func TestCompressFlushesBuffer(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Compress(middleware.CompressConfig{
		MinSize: 1000, // High threshold
		ContentTypes: []string{"text/plain"},
	}))

	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "small") // Under MinSize
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should still get the response even though it's under MinSize
	if rec.Body.String() != "small" {
		t.Errorf("expected 'small', got '%s'", rec.Body.String())
	}
}

// Test empty QueryParams
func TestQueryParamsEmpty(t *testing.T) {
	app := marten.New()

	app.GET("/", func(c *marten.Ctx) error {
		params := c.QueryParams()
		return c.OK(marten.M{"count": len(params)})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), `"count":0`) {
		t.Error("expected empty params")
	}
}

// Test Stream with large data
func TestContextStreamLarge(t *testing.T) {
	app := marten.New()

	largeData := strings.Repeat("x", 10000)

	app.GET("/", func(c *marten.Ctx) error {
		return c.Stream(200, "application/octet-stream", strings.NewReader(largeData))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.Len() != 10000 {
		t.Errorf("expected 10000 bytes, got %d", rec.Body.Len())
	}
}

// Test Stream with io.Reader that returns error
func TestContextStreamError(t *testing.T) {
	app := marten.New()

	app.GET("/", func(c *marten.Ctx) error {
		return c.Stream(200, "text/plain", &errorReader{})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should still return 200 (headers already written)
	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}


// Test CORS Vary header
func TestCORSVaryHeader(t *testing.T) {
	app := marten.New()
	app.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins: []string{"https://example.com"},
	}))
	app.GET("/", func(c *marten.Ctx) error {
		return c.OK(marten.M{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Vary") != "Origin" {
		t.Error("expected Vary: Origin header")
	}
}

// Test CORS ExposeHeaders
func TestCORSExposeHeaders(t *testing.T) {
	app := marten.New()
	app.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins:  []string{"*"},
		ExposeHeaders: []string{"X-Custom-Header", "X-Another"},
	}))
	app.GET("/", func(c *marten.Ctx) error {
		return c.OK(marten.M{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	expose := rec.Header().Get("Access-Control-Expose-Headers")
	if !strings.Contains(expose, "X-Custom-Header") {
		t.Error("expected X-Custom-Header in Expose-Headers")
	}
}

// Test CORS MaxAge
func TestCORSMaxAge(t *testing.T) {
	app := marten.New()
	app.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		MaxAge:       3600,
	}))
	app.GET("/", func(c *marten.Ctx) error {
		return c.OK(marten.M{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Max-Age") != "3600" {
		t.Errorf("expected Max-Age 3600, got %s", rec.Header().Get("Access-Control-Max-Age"))
	}
}

// Test ETag preserves response headers
func TestETagPreservesHeaders(t *testing.T) {
	app := marten.New()
	app.Use(middleware.ETag)
	app.GET("/", func(c *marten.Ctx) error {
		c.Header("X-Custom", "preserved")
		return c.Text(200, "hello world")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-Custom") != "preserved" {
		t.Error("ETag middleware should preserve custom headers")
	}
	if rec.Header().Get("ETag") == "" {
		t.Error("ETag header should be set")
	}
}

// Test BodyLimit with chunked encoding (no Content-Length)
func TestBodyLimitChunkedEncoding(t *testing.T) {
	app := marten.New()
	app.Use(middleware.BodyLimit(100))
	app.POST("/", func(c *marten.Ctx) error {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return c.BadRequest(err.Error())
		}
		return c.OK(marten.M{"size": len(body)})
	})

	// Small body should pass
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("small"))
	req.ContentLength = -1 // Simulate chunked encoding
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200 for small body, got %d", rec.Code)
	}
}


// Test 405 Method Not Allowed
func TestMethodNotAllowed(t *testing.T) {
	app := marten.New()

	app.GET("/users", func(c *marten.Ctx) error {
		return c.OK(marten.M{"users": []string{}})
	})
	app.POST("/users", func(c *marten.Ctx) error {
		return c.Created(marten.M{"id": 1})
	})

	// DELETE should return 405, not 404
	req := httptest.NewRequest(http.MethodDelete, "/users", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}

	allow := rec.Header().Get("Allow")
	if allow == "" {
		t.Error("expected Allow header")
	}
	if !strings.Contains(allow, "GET") || !strings.Contains(allow, "POST") {
		t.Errorf("Allow header should contain GET and POST, got: %s", allow)
	}
}

// Test 404 still works for non-existent paths
func TestNotFoundStillWorks(t *testing.T) {
	app := marten.New()

	app.GET("/users", func(c *marten.Ctx) error {
		return c.OK(marten.M{"users": []string{}})
	})

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}


// Test route conflict detection
func TestRouteConflictDetection(t *testing.T) {
	app := marten.New()

	// Register first route with :id param
	app.GET("/users/:id", func(c *marten.Ctx) error {
		return c.OK(marten.M{"id": c.Param("id")})
	})

	// Registering conflicting param should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for conflicting route params")
		}
	}()

	app.GET("/users/:name", func(c *marten.Ctx) error {
		return c.OK(marten.M{"name": c.Param("name")})
	})
}

// Test same param name is allowed
func TestSameParamNameAllowed(t *testing.T) {
	app := marten.New()

	// Same param name should not panic
	app.GET("/users/:id", func(c *marten.Ctx) error {
		return c.OK(marten.M{"id": c.Param("id")})
	})
	app.POST("/users/:id", func(c *marten.Ctx) error {
		return c.Created(marten.M{"id": c.Param("id")})
	})

	// Test both work
	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

// Test trailing slash ignore (default)
func TestTrailingSlashIgnore(t *testing.T) {
	app := marten.New()

	app.GET("/users", func(c *marten.Ctx) error {
		return c.OK(marten.M{"path": "users"})
	})

	// Without trailing slash
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200 for /users, got %d", rec.Code)
	}

	// With trailing slash should also work
	req = httptest.NewRequest(http.MethodGet, "/users/", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200 for /users/, got %d", rec.Code)
	}
}

// Test trailing slash redirect
func TestTrailingSlashRedirect(t *testing.T) {
	app := marten.New()
	app.SetTrailingSlash(marten.TrailingSlashRedirect)

	app.GET("/users", func(c *marten.Ctx) error {
		return c.OK(marten.M{"path": "users"})
	})

	// Request with trailing slash should redirect
	req := httptest.NewRequest(http.MethodGet, "/users/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 301 {
		t.Errorf("expected 301 redirect, got %d", rec.Code)
	}
	if rec.Header().Get("Location") != "/users" {
		t.Errorf("expected redirect to /users, got %s", rec.Header().Get("Location"))
	}
}

// Test trailing slash strict
func TestTrailingSlashStrict(t *testing.T) {
	app := marten.New()
	app.SetTrailingSlash(marten.TrailingSlashStrict)

	app.GET("/users", func(c *marten.Ctx) error {
		return c.OK(marten.M{"path": "users"})
	})

	// Without trailing slash works
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200 for /users, got %d", rec.Code)
	}

	// With trailing slash should 404
	req = httptest.NewRequest(http.MethodGet, "/users/", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Errorf("expected 404 for /users/ in strict mode, got %d", rec.Code)
	}
}

// Test trailing slash with params
func TestTrailingSlashWithParams(t *testing.T) {
	app := marten.New()

	app.GET("/users/:id", func(c *marten.Ctx) error {
		return c.OK(marten.M{"id": c.Param("id")})
	})

	// With trailing slash should work (ignore mode)
	req := httptest.NewRequest(http.MethodGet, "/users/123/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
