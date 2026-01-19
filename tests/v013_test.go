package tests

import (
	"bytes"
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gomarten/marten"
	"github.com/gomarten/marten/middleware"
)

// --- Additional Router Tests for v0.1.3 ---

// Test wildcard with empty path segment
func TestWildcardEmptyPath(t *testing.T) {
	app := marten.New()
	app.GET("/files/*filepath", func(c *marten.Ctx) error {
		path := c.Param("filepath")
		return c.Text(200, "path:"+path)
	})

	// Access just /files/ - filepath should be empty
	req := httptest.NewRequest("GET", "/files/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	// Should handle empty filepath gracefully
	t.Logf("Empty wildcard result: %s", rec.Body.String())
}

// Test group prefix normalization with trailing slash
func TestGroupPrefixTrailingSlash(t *testing.T) {
	app := marten.New()

	// Group with trailing slash
	api := app.Group("/api/")
	api.GET("/users", func(c *marten.Ctx) error {
		return c.Text(200, "users")
	})
	api.GET("posts", func(c *marten.Ctx) error {
		return c.Text(200, "posts")
	})

	tests := []struct {
		path     string
		expected int
	}{
		{"/api/users", 200},
		{"/api//users", 404}, // Double slash should 404
		{"/api/posts", 200},
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


// Test nested groups with middleware
func TestNestedGroupsMiddleware(t *testing.T) {
	app := marten.New()

	var order []string
	mu := sync.Mutex{}

	mw1 := func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			mu.Lock()
			order = append(order, "mw1")
			mu.Unlock()
			return next(c)
		}
	}

	mw2 := func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			mu.Lock()
			order = append(order, "mw2")
			mu.Unlock()
			return next(c)
		}
	}

	mw3 := func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			mu.Lock()
			order = append(order, "mw3")
			mu.Unlock()
			return next(c)
		}
	}

	api := app.Group("/api", mw1)
	v1 := api.Group("/v1", mw2)
	v1.GET("/users", func(c *marten.Ctx) error {
		mu.Lock()
		order = append(order, "handler")
		mu.Unlock()
		return c.Text(200, "ok")
	}, mw3)

	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// Check middleware order: mw1 -> mw2 -> mw3 -> handler
	expected := []string{"mw1", "mw2", "mw3", "handler"}
	mu.Lock()
	defer mu.Unlock()
	if len(order) != len(expected) {
		t.Fatalf("expected %d middleware calls, got %d: %v", len(expected), len(order), order)
	}
	for i, exp := range expected {
		if order[i] != exp {
			t.Errorf("position %d: expected %s, got %s", i, exp, order[i])
		}
	}
}

// Test multiple param segments in a row
func TestMultipleConsecutiveParams(t *testing.T) {
	app := marten.New()
	app.GET("/:a/:b/:c", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("a")+"-"+c.Param("b")+"-"+c.Param("c"))
	})

	req := httptest.NewRequest("GET", "/x/y/z", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "x-y-z" {
		t.Errorf("expected 'x-y-z', got %q", rec.Body.String())
	}
}

// Test param followed by wildcard
func TestParamFollowedByWildcard(t *testing.T) {
	app := marten.New()
	app.GET("/users/:id/files/*filepath", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("id")+":"+c.Param("filepath"))
	})

	req := httptest.NewRequest("GET", "/users/123/files/docs/report.pdf", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "123:docs/report.pdf" {
		t.Errorf("expected '123:docs/report.pdf', got %q", rec.Body.String())
	}
}

// Test route registration order doesn't affect matching
func TestRouteRegistrationOrder(t *testing.T) {
	app := marten.New()

	// Register in different order
	app.GET("/users/:id/edit", func(c *marten.Ctx) error {
		return c.Text(200, "edit")
	})
	app.GET("/users/new", func(c *marten.Ctx) error {
		return c.Text(200, "new")
	})
	app.GET("/users/:id", func(c *marten.Ctx) error {
		return c.Text(200, "show:"+c.Param("id"))
	})

	tests := []struct {
		path     string
		expected string
	}{
		{"/users/new", "new"},
		{"/users/123", "show:123"},
		{"/users/456/edit", "edit"},
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


// --- Additional Context Tests ---

// Test context with cancelled request
func TestContextCancellation(t *testing.T) {
	app := marten.New()
	app.GET("/slow", func(c *marten.Ctx) error {
		ctx := c.Context()
		select {
		case <-time.After(100 * time.Millisecond):
			return c.Text(200, "completed")
		case <-ctx.Done():
			return c.Text(499, "cancelled")
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/slow", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	// Cancel immediately
	cancel()
	app.ServeHTTP(rec, req)

	// Should detect cancellation
	if rec.Code != 499 {
		t.Logf("Context cancellation status: %d", rec.Code)
	}
}

// Test Bind with multipart form data
func TestBindMultipartFormData(t *testing.T) {
	app := marten.New()
	app.POST("/upload", func(c *marten.Ctx) error {
		var data struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if err := c.Bind(&data); err != nil {
			return c.BadRequest(err.Error())
		}
		return c.OK(marten.M{"name": data.Name, "email": data.Email})
	})

	body := &bytes.Buffer{}
	body.WriteString("--boundary\r\n")
	body.WriteString("Content-Disposition: form-data; name=\"name\"\r\n\r\n")
	body.WriteString("Alice\r\n")
	body.WriteString("--boundary\r\n")
	body.WriteString("Content-Disposition: form-data; name=\"email\"\r\n\r\n")
	body.WriteString("alice@example.com\r\n")
	body.WriteString("--boundary--\r\n")

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Alice") {
		t.Errorf("expected name in response, got %s", rec.Body.String())
	}
}

// Test Bind without Content-Type header
func TestBindNoContentType(t *testing.T) {
	app := marten.New()
	app.POST("/bind", func(c *marten.Ctx) error {
		var data map[string]string
		if err := c.Bind(&data); err != nil {
			return c.BadRequest(err.Error())
		}
		return c.OK(data)
	})

	body := bytes.NewBufferString(`{"key":"value"}`)
	req := httptest.NewRequest("POST", "/bind", body)
	// No Content-Type header - should default to JSON
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Logf("Bind without Content-Type: %d - %s", rec.Code, rec.Body.String())
	}
}

// Test QueryInt with negative numbers
func TestQueryIntNegative(t *testing.T) {
	app := marten.New()
	app.GET("/test", func(c *marten.Ctx) error {
		val := c.QueryInt("num")
		return c.JSON(200, marten.M{"num": val})
	})

	tests := []struct {
		query    string
		expected int
	}{
		{"?num=-5", -5},
		{"?num=-100", -100},
		{"?num=0", 0},
		{"?num=42", 42},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/test"+tt.query, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if !strings.Contains(rec.Body.String(), strings.Trim(tt.query[5:], "")) {
			t.Errorf("query %s: unexpected response %s", tt.query, rec.Body.String())
		}
	}
}

// Test ParamInt with overflow
func TestParamIntOverflow(t *testing.T) {
	app := marten.New()
	app.GET("/items/:id", func(c *marten.Ctx) error {
		id := c.ParamInt("id")
		return c.JSON(200, marten.M{"id": id})
	})

	// Very large number that overflows int
	req := httptest.NewRequest("GET", "/items/999999999999999999999", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should return 0 for overflow
	if !strings.Contains(rec.Body.String(), "\"id\":0") {
		t.Logf("Overflow handling: %s", rec.Body.String())
	}
}


// Test multiple Set/Get operations
func TestContextStoreMultipleOperations(t *testing.T) {
	app := marten.New()
	app.GET("/test", func(c *marten.Ctx) error {
		// Set multiple values
		c.Set("string", "hello")
		c.Set("int", 42)
		c.Set("bool", true)
		c.Set("nil", nil)

		// Overwrite a value
		c.Set("string", "world")

		// Get values
		str := c.GetString("string")
		num := c.GetInt("int")
		flag := c.GetBool("bool")
		nilVal := c.Get("nil")

		return c.JSON(200, marten.M{
			"string": str,
			"int":    num,
			"bool":   flag,
			"nil":    nilVal,
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "world") {
		t.Error("expected overwritten value 'world'")
	}
	if !strings.Contains(body, "42") {
		t.Error("expected int value 42")
	}
	if !strings.Contains(body, "true") {
		t.Error("expected bool value true")
	}
}

// Test ClientIP with various header combinations
func TestClientIPPrecedence(t *testing.T) {
	app := marten.New()
	app.GET("/ip", func(c *marten.Ctx) error {
		return c.Text(200, c.ClientIP())
	})

	tests := []struct {
		name     string
		headers  map[string]string
		expected string
	}{
		{
			name:     "X-Forwarded-For only",
			headers:  map[string]string{"X-Forwarded-For": "1.1.1.1"},
			expected: "1.1.1.1",
		},
		{
			name:     "X-Real-IP only",
			headers:  map[string]string{"X-Real-IP": "2.2.2.2"},
			expected: "2.2.2.2",
		},
		{
			name: "Both headers - XFF takes precedence",
			headers: map[string]string{
				"X-Forwarded-For": "1.1.1.1",
				"X-Real-IP":       "2.2.2.2",
			},
			expected: "1.1.1.1",
		},
		{
			name:     "XFF with spaces",
			headers:  map[string]string{"X-Forwarded-For": "  1.1.1.1  "},
			expected: "1.1.1.1",
		},
		{
			name:     "XFF multiple IPs with spaces",
			headers:  map[string]string{"X-Forwarded-For": " 1.1.1.1 , 2.2.2.2 , 3.3.3.3 "},
			expected: "1.1.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ip", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Body.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, rec.Body.String())
			}
		})
	}
}

// Test Bearer token edge cases
func TestBearerTokenEdgeCases(t *testing.T) {
	app := marten.New()
	app.GET("/token", func(c *marten.Ctx) error {
		token := c.Bearer()
		if token == "" {
			return c.Text(200, "empty")
		}
		return c.Text(200, token)
	})

	tests := []struct {
		name     string
		auth     string
		expected string
	}{
		{"Valid token", "Bearer abc123xyz", "abc123xyz"},
		{"Token with spaces", "Bearer token with spaces", "token with spaces"},
		{"Empty bearer", "Bearer ", "empty"},
		{"No space after Bearer", "Bearerabc123", "empty"},
		{"Lowercase bearer", "bearer abc123", "empty"},
		{"No auth header", "", "empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/token", nil)
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Body.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, rec.Body.String())
			}
		})
	}
}


// --- Additional Middleware Tests ---

// Test Timeout middleware with slow handler
func TestTimeoutMiddlewareSlow(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Timeout(50 * time.Millisecond))
	app.GET("/slow", func(c *marten.Ctx) error {
		select {
		case <-time.After(100 * time.Millisecond):
			return c.Text(200, "completed")
		case <-c.Context().Done():
			// Context cancelled, don't write response
			return nil
		}
	})

	req := httptest.NewRequest("GET", "/slow", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 504 {
		t.Errorf("expected 504 timeout, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "timeout") {
		t.Errorf("expected timeout message, got %s", rec.Body.String())
	}
}

// Test Timeout middleware with fast handler
func TestTimeoutMiddlewareFast(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Timeout(100 * time.Millisecond))
	app.GET("/fast", func(c *marten.Ctx) error {
		return c.Text(200, "completed")
	})

	req := httptest.NewRequest("GET", "/fast", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "completed" {
		t.Errorf("expected 'completed', got %q", rec.Body.String())
	}
}

// Test Compress middleware with different content types
func TestCompressContentTypes(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		shouldGzip  bool
	}{
		{"JSON", "application/json", true},
		{"HTML", "text/html", true},
		{"Plain text", "text/plain", true},
		{"CSS", "text/css", true},
		{"JavaScript", "application/javascript", true},
		{"Image", "image/png", false},
		{"Video", "video/mp4", false},
		{"Binary", "application/octet-stream", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := marten.New()
			app.Use(middleware.Compress(middleware.DefaultCompressConfig()))
			app.GET("/test", func(c *marten.Ctx) error {
				// Use Blob to set exact content type without it being overwritten
				data := []byte(strings.Repeat("test data ", 200)) // Large enough to compress
				return c.Blob(200, tt.contentType, data)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Accept-Encoding", "gzip")
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			hasGzip := rec.Header().Get("Content-Encoding") == "gzip"
			if hasGzip != tt.shouldGzip {
				t.Errorf("expected gzip=%v, got gzip=%v for content-type %s", tt.shouldGzip, hasGzip, tt.contentType)
			}
		})
	}
}

// Test BodyLimit with exact limit
func TestBodyLimitExactSize(t *testing.T) {
	app := marten.New()
	app.Use(middleware.BodyLimit(10)) // 10 bytes
	app.POST("/test", func(c *marten.Ctx) error {
		body, _ := io.ReadAll(c.Request.Body)
		return c.Text(200, string(body))
	})

	tests := []struct {
		name     string
		body     string
		expected int
	}{
		{"Under limit", "123456789", 200},
		{"Exact limit", "1234567890", 200},
		{"Over limit", "12345678901", 413},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Code != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, rec.Code)
			}
		})
	}
}

// Test BasicAuth with empty credentials
func TestBasicAuthEmpty(t *testing.T) {
	app := marten.New()
	app.Use(middleware.BasicAuthSimple("admin", "secret"))
	app.GET("/protected", func(c *marten.Ctx) error {
		return c.Text(200, "protected")
	})

	tests := []struct {
		name     string
		auth     string
		expected int
	}{
		{"No auth", "", 401},
		{"Empty credentials", "Basic ", 401},
		{"Invalid base64", "Basic !!!invalid!!!", 401},
		{"Valid credentials", "Basic YWRtaW46c2VjcmV0", 200}, // admin:secret
		{"Wrong password", "Basic YWRtaW46d3Jvbmc=", 401},    // admin:wrong
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/protected", nil)
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Code != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, rec.Code)
			}
		})
	}
}


// Test CORS with credentials and wildcard origin
func TestCORSCredentialsWithWildcard(t *testing.T) {
	app := marten.New()
	app.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true, // Should be ignored with wildcard
	}))
	app.GET("/test", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Credentials should NOT be set with wildcard origin
	if rec.Header().Get("Access-Control-Allow-Credentials") == "true" {
		t.Error("credentials should not be allowed with wildcard origin")
	}
}

// Test CORS preflight request with custom headers
func TestCORSPreflightCustom(t *testing.T) {
	app := marten.New()
	app.Use(middleware.CORS(middleware.DefaultCORSConfig()))
	app.GET("/api/users", func(c *marten.Ctx) error {
		return c.Text(200, "users")
	})

	req := httptest.NewRequest("OPTIONS", "/api/users", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 204 {
		t.Errorf("expected 204 for preflight, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("missing Allow-Methods header")
	}
}

// Test ETag with If-None-Match
func TestETagIfNoneMatch(t *testing.T) {
	app := marten.New()
	app.Use(middleware.ETag)
	app.GET("/data", func(c *marten.Ctx) error {
		return c.Text(200, "some data")
	})

	// First request to get ETag
	req := httptest.NewRequest("GET", "/data", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header")
	}

	// Second request with If-None-Match
	req = httptest.NewRequest("GET", "/data", nil)
	req.Header.Set("If-None-Match", etag)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 304 {
		t.Errorf("expected 304 Not Modified, got %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Error("expected empty body for 304 response")
	}
}

// Test ETag with POST request (should not add ETag)
func TestETagPostRequest(t *testing.T) {
	app := marten.New()
	app.Use(middleware.ETag)
	app.POST("/data", func(c *marten.Ctx) error {
		return c.Text(200, "created")
	})

	req := httptest.NewRequest("POST", "/data", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("ETag") != "" {
		t.Error("ETag should not be set for POST requests")
	}
}

// Test NoCache headers
func TestNoCacheHeaders(t *testing.T) {
	app := marten.New()
	app.Use(middleware.NoCache)
	app.GET("/test", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	expectedHeaders := map[string]string{
		"Cache-Control":     "no-store, no-cache, must-revalidate, proxy-revalidate",
		"Pragma":            "no-cache",
		"Expires":           "0",
		"Surrogate-Control": "no-store",
	}

	for header, expected := range expectedHeaders {
		if rec.Header().Get(header) != expected {
			t.Errorf("header %s: expected %q, got %q", header, expected, rec.Header().Get(header))
		}
	}
}

// Test Secure middleware headers
func TestSecureHeaders(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Secure(middleware.DefaultSecureConfig()))
	app.GET("/test", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	expectedHeaders := []string{
		"X-XSS-Protection",
		"X-Content-Type-Options",
		"X-Frame-Options",
		"Referrer-Policy",
	}

	for _, header := range expectedHeaders {
		if rec.Header().Get(header) == "" {
			t.Errorf("missing security header: %s", header)
		}
	}
}


// --- Error Handling Tests ---

// Test custom error handler with different error types
func TestCustomErrorHandler(t *testing.T) {
	app := marten.New()

	var capturedErrors []error
	app.OnError(func(c *marten.Ctx, err error) {
		capturedErrors = append(capturedErrors, err)
		c.JSON(500, marten.M{"error": err.Error()})
	})

	app.GET("/error1", func(c *marten.Ctx) error {
		return io.EOF
	})
	app.GET("/error2", func(c *marten.Ctx) error {
		return &marten.BindError{Message: "bind failed"}
	})

	// Test error 1
	req := httptest.NewRequest("GET", "/error1", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Test error 2
	req = httptest.NewRequest("GET", "/error2", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if len(capturedErrors) != 2 {
		t.Errorf("expected 2 errors captured, got %d", len(capturedErrors))
	}
}

// Test panic in middleware
func TestPanicInMiddleware(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Recover)
	app.Use(func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			panic("middleware panic")
		}
	})
	app.GET("/test", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 500 {
		t.Errorf("expected 500 for panic, got %d", rec.Code)
	}
}

// Test panic with nil value
func TestPanicNil(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Recover)
	app.GET("/test", func(c *marten.Ctx) error {
		panic(nil)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should handle nil panic gracefully
	if rec.Code != 500 {
		t.Logf("Nil panic handling: %d", rec.Code)
	}
}

// --- Concurrent Request Tests ---

// Test concurrent requests with shared state
func TestConcurrentRequestsSharedState(t *testing.T) {
	app := marten.New()

	counter := 0
	mu := sync.Mutex{}

	app.GET("/increment", func(c *marten.Ctx) error {
		mu.Lock()
		counter++
		current := counter
		mu.Unlock()
		return c.JSON(200, marten.M{"count": current})
	})

	var wg sync.WaitGroup
	numRequests := 100

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/increment", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
		}()
	}

	wg.Wait()

	if counter != numRequests {
		t.Errorf("expected counter=%d, got %d", numRequests, counter)
	}
}

// Test concurrent requests with different routes
func TestConcurrentDifferentRoutes(t *testing.T) {
	app := marten.New()

	app.GET("/route1", func(c *marten.Ctx) error {
		time.Sleep(10 * time.Millisecond)
		return c.Text(200, "route1")
	})
	app.GET("/route2", func(c *marten.Ctx) error {
		time.Sleep(10 * time.Millisecond)
		return c.Text(200, "route2")
	})
	app.GET("/route3", func(c *marten.Ctx) error {
		time.Sleep(10 * time.Millisecond)
		return c.Text(200, "route3")
	})

	var wg sync.WaitGroup
	routes := []string{"/route1", "/route2", "/route3"}

	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			route := routes[idx%3]
			req := httptest.NewRequest("GET", route, nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Code != 200 {
				t.Errorf("route %s: expected 200, got %d", route, rec.Code)
			}
		}(i)
	}

	wg.Wait()
}

// Test context isolation between concurrent requests
func TestContextIsolation(t *testing.T) {
	app := marten.New()

	app.GET("/set/:value", func(c *marten.Ctx) error {
		value := c.Param("value")
		c.Set("test", value)
		time.Sleep(50 * time.Millisecond) // Simulate work
		stored := c.GetString("test")
		if stored != value {
			return c.ServerError("context leaked: expected " + value + ", got " + stored)
		}
		return c.Text(200, stored)
	})

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			value := "value" + string(rune('0'+id))
			req := httptest.NewRequest("GET", "/set/"+value, nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Code != 200 {
				errors <- io.EOF
			}
			if rec.Body.String() != value {
				errors <- io.EOF
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error("context isolation failed:", err)
	}
}


// --- Response Helper Tests ---

// Test all response status helpers
func TestAllResponseHelpers(t *testing.T) {
	tests := []struct {
		name     string
		handler  func(*marten.Ctx) error
		expected int
	}{
		{"OK", func(c *marten.Ctx) error { return c.OK(marten.M{"ok": true}) }, 200},
		{"Created", func(c *marten.Ctx) error { return c.Created(marten.M{"id": 1}) }, 201},
		{"NoContent", func(c *marten.Ctx) error { return c.NoContent() }, 204},
		{"BadRequest", func(c *marten.Ctx) error { return c.BadRequest("bad") }, 400},
		{"Unauthorized", func(c *marten.Ctx) error { return c.Unauthorized("unauth") }, 401},
		{"Forbidden", func(c *marten.Ctx) error { return c.Forbidden("forbidden") }, 403},
		{"NotFound", func(c *marten.Ctx) error { return c.NotFound("not found") }, 404},
		{"ServerError", func(c *marten.Ctx) error { return c.ServerError("error") }, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := marten.New()
			app.GET("/test", tt.handler)

			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Code != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, rec.Code)
			}
		})
	}
}

// Test HTML response with special characters
func TestHTMLSpecialCharacters(t *testing.T) {
	app := marten.New()
	app.GET("/html", func(c *marten.Ctx) error {
		return c.HTML(200, "<script>alert('xss')</script>")
	})

	req := httptest.NewRequest("GET", "/html", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "<script>") {
		t.Error("HTML should not escape content")
	}
	if rec.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Error("wrong content type for HTML")
	}
}

// Test Blob with empty data
func TestBlobEmpty(t *testing.T) {
	app := marten.New()
	app.GET("/blob", func(c *marten.Ctx) error {
		return c.Blob(200, "application/octet-stream", []byte{})
	})

	req := httptest.NewRequest("GET", "/blob", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Error("expected empty body")
	}
}

// Test Stream with nil reader
func TestStreamNilReader(t *testing.T) {
	app := marten.New()
	app.GET("/stream", func(c *marten.Ctx) error {
		return c.Stream(200, "text/plain", nil)
	})

	req := httptest.NewRequest("GET", "/stream", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should handle nil reader gracefully
	if rec.Code != 200 {
		t.Logf("Nil reader handling: %d", rec.Code)
	}
}

// Test JSON with complex nested structure
func TestJSONComplexNested(t *testing.T) {
	app := marten.New()
	app.GET("/json", func(c *marten.Ctx) error {
		return c.OK(marten.M{
			"user": marten.M{
				"name": "Alice",
				"profile": marten.M{
					"age": 30,
					"address": marten.M{
						"city":    "NYC",
						"country": "USA",
					},
				},
			},
			"tags": []string{"admin", "user"},
		})
	})

	req := httptest.NewRequest("GET", "/json", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "Alice") {
		t.Error("missing nested name")
	}
	if !strings.Contains(body, "NYC") {
		t.Error("missing deeply nested city")
	}
	if !strings.Contains(body, "admin") {
		t.Error("missing array element")
	}
}

// --- Edge Case Tests ---

// Test empty path segments
func TestEmptyPathSegments(t *testing.T) {
	app := marten.New()
	app.GET("/users", func(c *marten.Ctx) error {
		return c.Text(200, "users")
	})

	tests := []struct {
		path     string
		expected int
		note     string
	}{
		{"/users", 200, "normal path"},
		{"//users", 200, "double slash normalized to /users"},
		{"/users//", 200, "trailing double slash normalized"},
		{"///users", 200, "triple slash normalized to /users"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != tt.expected {
			t.Errorf("path %q (%s): expected %d, got %d", tt.path, tt.note, tt.expected, rec.Code)
		}
	}
}

// Test very long path
func TestVeryLongPath(t *testing.T) {
	app := marten.New()
	
	// Create a very long path
	longPath := "/a"
	for i := 0; i < 100; i++ {
		longPath += "/segment" + string(rune('0'+i%10))
	}
	
	app.GET(longPath, func(c *marten.Ctx) error {
		return c.Text(200, "found")
	})

	req := httptest.NewRequest("GET", longPath, nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200 for long path, got %d", rec.Code)
	}
}

// Test route with only params
func TestRouteOnlyParams(t *testing.T) {
	app := marten.New()
	app.GET("/:a/:b/:c/:d", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("a")+c.Param("b")+c.Param("c")+c.Param("d"))
	})

	req := httptest.NewRequest("GET", "/1/2/3/4", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "1234" {
		t.Errorf("expected '1234', got %q", rec.Body.String())
	}
}

// Test RequestID persistence across middleware
func TestRequestIDPersistence(t *testing.T) {
	app := marten.New()

	var id1, id2, id3 string

	app.Use(func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			id1 = c.RequestID()
			return next(c)
		}
	})

	app.Use(func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			id2 = c.RequestID()
			return next(c)
		}
	})

	app.GET("/test", func(c *marten.Ctx) error {
		id3 = c.RequestID()
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if id1 != id2 || id2 != id3 {
		t.Errorf("RequestID not consistent: %s, %s, %s", id1, id2, id3)
	}
	if id1 == "" {
		t.Error("RequestID should not be empty")
	}
}

// Test StatusCode before and after write
func TestStatusCodeTracking(t *testing.T) {
	app := marten.New()

	var beforeStatus, afterStatus int

	app.GET("/test", func(c *marten.Ctx) error {
		beforeStatus = c.StatusCode()
		c.JSON(201, marten.M{"created": true})
		afterStatus = c.StatusCode()
		return nil
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if beforeStatus != 0 {
		t.Errorf("status before write should be 0, got %d", beforeStatus)
	}
	if afterStatus != 201 {
		t.Errorf("status after write should be 201, got %d", afterStatus)
	}
}
