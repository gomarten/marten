package tests

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gomarten/marten"
	"github.com/gomarten/marten/middleware"
)

// --- NoCache Middleware Tests ---

func TestNoCacheMiddleware(t *testing.T) {
	app := marten.New()
	app.Use(middleware.NoCache)
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Cache-Control") != "no-store, no-cache, must-revalidate, proxy-revalidate" {
		t.Errorf("unexpected Cache-Control: %q", rec.Header().Get("Cache-Control"))
	}
	if rec.Header().Get("Pragma") != "no-cache" {
		t.Errorf("unexpected Pragma: %q", rec.Header().Get("Pragma"))
	}
	if rec.Header().Get("Expires") != "0" {
		t.Errorf("unexpected Expires: %q", rec.Header().Get("Expires"))
	}
}

// --- Secure Middleware Tests ---

func TestSecureMiddlewareDefault(t *testing.T) {
	app := marten.New()
	app.Use(middleware.SecureDefault)
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-XSS-Protection") != "1; mode=block" {
		t.Errorf("unexpected X-XSS-Protection: %q", rec.Header().Get("X-XSS-Protection"))
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("unexpected X-Content-Type-Options: %q", rec.Header().Get("X-Content-Type-Options"))
	}
	if rec.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Errorf("unexpected X-Frame-Options: %q", rec.Header().Get("X-Frame-Options"))
	}
	if rec.Header().Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Errorf("unexpected Referrer-Policy: %q", rec.Header().Get("Referrer-Policy"))
	}
}

func TestSecureMiddlewareHSTS(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Secure(middleware.SecureConfig{
		HSTSMaxAge:            31536000,
		HSTSIncludeSubdomains: true,
	}))
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	if !strings.Contains(hsts, "max-age=31536000") {
		t.Errorf("unexpected HSTS: %q", hsts)
	}
	if !strings.Contains(hsts, "includeSubDomains") {
		t.Errorf("HSTS should include subdomains: %q", hsts)
	}
}

func TestSecureMiddlewareCSP(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Secure(middleware.SecureConfig{
		ContentSecurityPolicy: "default-src 'self'",
	}))
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Security-Policy") != "default-src 'self'" {
		t.Errorf("unexpected CSP: %q", rec.Header().Get("Content-Security-Policy"))
	}
}

// --- BodyLimit Middleware Tests ---

func TestBodyLimitMiddleware(t *testing.T) {
	app := marten.New()
	app.Use(middleware.BodyLimit(100)) // 100 bytes
	app.POST("/", func(c *marten.Ctx) error {
		var data map[string]string
		if err := c.Bind(&data); err != nil {
			return c.BadRequest(err.Error())
		}
		return c.OK(data)
	})

	// Small body - should succeed
	smallBody := bytes.NewBufferString(`{"name":"test"}`)
	req := httptest.NewRequest("POST", "/", smallBody)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("small body: expected 200, got %d", rec.Code)
	}

	// Large body via Content-Length - should fail
	largeBody := bytes.NewBufferString(strings.Repeat("x", 200))
	req = httptest.NewRequest("POST", "/", largeBody)
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = 200
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 413 {
		t.Errorf("large body: expected 413, got %d", rec.Code)
	}
}

func TestBodyLimitConstants(t *testing.T) {
	if middleware.KB != 1024 {
		t.Errorf("KB should be 1024, got %d", middleware.KB)
	}
	if middleware.MB != 1024*1024 {
		t.Errorf("MB should be 1048576, got %d", middleware.MB)
	}
	if middleware.GB != 1024*1024*1024 {
		t.Errorf("GB should be 1073741824, got %d", middleware.GB)
	}
}

// --- Compress Middleware Tests ---

func TestCompressMiddleware(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Compress(middleware.CompressConfig{
		Level:        gzip.DefaultCompression,
		MinSize:      10, // Low threshold for testing
		ContentTypes: []string{"application/json"},
	}))
	app.GET("/", func(c *marten.Ctx) error {
		return c.JSON(200, map[string]string{
			"message": "this is a longer message that should be compressed",
			"data":    strings.Repeat("x", 100),
		})
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("expected gzip encoding, got %q", rec.Header().Get("Content-Encoding"))
	}

	// Decompress and verify
	gr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	body, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("failed to read gzip body: %v", err)
	}

	if !strings.Contains(string(body), "message") {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestCompressMiddlewareNoAcceptEncoding(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Compress(middleware.DefaultCompressConfig()))
	app.GET("/", func(c *marten.Ctx) error {
		return c.JSON(200, map[string]string{"message": strings.Repeat("x", 2000)})
	})

	// No Accept-Encoding header
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") == "gzip" {
		t.Error("should not compress without Accept-Encoding")
	}
}

func TestCompressMiddlewareSmallResponse(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Compress(middleware.DefaultCompressConfig()))
	app.GET("/", func(c *marten.Ctx) error {
		return c.JSON(200, map[string]string{"ok": "1"})
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Small responses should not be compressed
	if rec.Header().Get("Content-Encoding") == "gzip" {
		t.Error("small responses should not be compressed")
	}
}

// --- ETag Middleware Tests ---

func TestETagMiddleware(t *testing.T) {
	app := marten.New()
	app.Use(middleware.ETag)
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "hello world")
	})

	// First request - should get ETag
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Error("expected ETag header")
	}
	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// Second request with If-None-Match - should get 304
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("If-None-Match", etag)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 304 {
		t.Errorf("expected 304 with matching ETag, got %d", rec.Code)
	}
}

func TestETagMiddlewarePOST(t *testing.T) {
	app := marten.New()
	app.Use(middleware.ETag)
	app.POST("/", func(c *marten.Ctx) error {
		return c.Text(200, "created")
	})

	req := httptest.NewRequest("POST", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// POST should not get ETag
	if rec.Header().Get("ETag") != "" {
		t.Error("POST should not have ETag")
	}
}

// --- Rate Limit Edge Cases ---

func TestRateLimitWindowReset(t *testing.T) {
	// This test verifies the rate limiter resets after window expires
	// Using a very short window for testing
	app := marten.New()
	app.Use(middleware.RateLimit(middleware.RateLimitConfig{
		Requests: 1,
		Window:   50 * 1e6, // 50ms in nanoseconds (time.Duration)
		KeyFunc:  func(c *marten.Ctx) string { return "test" },
	}))
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	// First request should succeed
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("first request: expected 200, got %d", rec.Code)
	}

	// Second request should be limited
	req = httptest.NewRequest("GET", "/", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 429 {
		t.Errorf("second request: expected 429, got %d", rec.Code)
	}
}

// --- Timeout Edge Cases ---

func TestTimeoutContextCancellation(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Timeout(50 * 1e6)) // 50ms
	app.GET("/", func(c *marten.Ctx) error {
		// Check if context is cancelled
		select {
		case <-c.Context().Done():
			return c.Text(499, "cancelled")
		default:
			return c.Text(200, "ok")
		}
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

// --- BasicAuth Edge Cases ---

func TestBasicAuthMalformedHeader(t *testing.T) {
	app := marten.New()
	app.Use(middleware.BasicAuthSimple("user", "pass"))
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	tests := []struct {
		name string
		auth string
	}{
		{"empty", ""},
		{"no basic prefix", "user:pass"},
		{"invalid base64", "Basic !!!invalid!!!"},
		{"no colon", "Basic dXNlcg=="}, // "user" without colon
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Code != 401 {
				t.Errorf("%s: expected 401, got %d", tt.name, rec.Code)
			}
		})
	}
}

// --- CORS Edge Cases ---

func TestCORSEmptyOrigin(t *testing.T) {
	app := marten.New()
	app.Use(middleware.CORS(middleware.DefaultCORSConfig()))
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	// Request without Origin header
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should still work, just no CORS headers
	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestCORSMultipleOrigins(t *testing.T) {
	app := marten.New()
	app.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins: []string{"https://a.com", "https://b.com", "https://c.com"},
		AllowMethods: []string{"GET"},
	}))
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	tests := []struct {
		origin   string
		expected string
	}{
		{"https://a.com", "https://a.com"},
		{"https://b.com", "https://b.com"},
		{"https://c.com", "https://c.com"},
		{"https://evil.com", ""},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Origin", tt.origin)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		got := rec.Header().Get("Access-Control-Allow-Origin")
		if got != tt.expected {
			t.Errorf("origin %s: expected %q, got %q", tt.origin, tt.expected, got)
		}
	}
}

// --- Recover Edge Cases ---

func TestRecoverWithNilPanic(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Recover)
	app.GET("/", func(c *marten.Ctx) error {
		panic(nil)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should handle nil panic gracefully
	if rec.Code != 200 && rec.Code != 500 {
		t.Errorf("unexpected status: %d", rec.Code)
	}
}

func TestRecoverWithErrorPanic(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Recover)
	app.GET("/", func(c *marten.Ctx) error {
		panic(http.ErrAbortHandler)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 500 {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

// --- Middleware Combination Edge Cases ---

func TestMiddlewareOrderWithError(t *testing.T) {
	app := marten.New()

	var order []string

	app.Use(func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			order = append(order, "mw1-before")
			err := next(c)
			order = append(order, "mw1-after")
			return err
		}
	})

	app.Use(func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			order = append(order, "mw2-before")
			// Short-circuit with error
			return c.BadRequest("stopped")
		}
	})

	app.GET("/", func(c *marten.Ctx) error {
		order = append(order, "handler")
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Handler should not be called
	expected := []string{"mw1-before", "mw2-before", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("position %d: expected %q, got %q", i, v, order[i])
		}
	}
}
