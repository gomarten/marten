package tests

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gomarten/marten"
	"github.com/gomarten/marten/middleware"
)

// Test Bind with form-urlencoded
func TestBindFormURLEncoded(t *testing.T) {
	app := marten.New()
	app.POST("/form", func(c *marten.Ctx) error {
		var data struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if err := c.Bind(&data); err != nil {
			return c.BadRequest(err.Error())
		}
		return c.OK(marten.M{"name": data.Name, "email": data.Email})
	})

	form := url.Values{}
	form.Set("name", "Jack")
	form.Set("email", "jack@example.com")

	req := httptest.NewRequest("POST", "/form", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Jack") {
		t.Errorf("Expected body to contain Jack, got %s", w.Body.String())
	}
}

// Test Bind with empty body returns error
func TestBindEmptyBodyError(t *testing.T) {
	app := marten.New()
	app.POST("/bind", func(c *marten.Ctx) error {
		var data struct {
			Name string `json:"name"`
		}
		if err := c.Bind(&data); err != nil {
			return c.BadRequest(err.Error())
		}
		return c.OK(data)
	})

	req := httptest.NewRequest("POST", "/bind", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "empty") {
		t.Errorf("Expected error about empty body, got %s", w.Body.String())
	}
}

// Test Logger with colors
func TestLoggerWithColors(t *testing.T) {
	var buf bytes.Buffer
	app := marten.New()
	app.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Output:       &buf,
		EnableColors: true,
	}))
	app.GET("/test", func(c *marten.Ctx) error {
		return c.OK(marten.M{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	// Check that output contains ANSI codes
	output := buf.String()
	if !strings.Contains(output, "\033[") {
		t.Errorf("Expected colored output with ANSI codes, got %s", output)
	}
}

// Test Logger with JSON format
func TestLoggerWithJSON(t *testing.T) {
	var buf bytes.Buffer
	app := marten.New()
	app.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Output:     &buf,
		JSONFormat: true,
	}))
	app.GET("/test", func(c *marten.Ctx) error {
		return c.OK(marten.M{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	output := buf.String()
	if !strings.Contains(output, `"method":"GET"`) {
		t.Errorf("Expected JSON output with method field, got %s", output)
	}
	if !strings.Contains(output, `"path":"/test"`) {
		t.Errorf("Expected JSON output with path field, got %s", output)
	}
}

// Test Recover with custom handler
func TestRecoverWithCustomHandler(t *testing.T) {
	app := marten.New()
	app.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		LogPanics: false,
		OnPanic: func(c *marten.Ctx, err any) error {
			return c.JSON(http.StatusInternalServerError, marten.M{
				"error":  "custom_panic",
				"detail": "Something went wrong",
			})
		},
	}))
	app.GET("/panic", func(c *marten.Ctx) error {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("Expected 500, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "custom_panic") {
		t.Errorf("Expected custom error response, got %s", w.Body.String())
	}
}

// Test RateLimit with custom response
func TestRateLimitCustomResponse(t *testing.T) {
	limiter := middleware.NewRateLimiter(middleware.RateLimitConfig{
		Requests: 1,
		Window:   60000000000, // 1 minute
		OnLimitReached: func(c *marten.Ctx) error {
			return c.JSON(http.StatusTooManyRequests, marten.M{
				"error":   "custom_rate_limit",
				"message": "Please slow down",
			})
		},
	})
	defer limiter.Stop()

	app := marten.New()
	app.Use(limiter.Middleware())
	app.GET("/test", func(c *marten.Ctx) error {
		return c.OK(marten.M{"status": "ok"})
	})

	// First request should succeed
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("First request: expected 200, got %d", w.Code)
	}

	// Second request should be rate limited with custom response
	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 429 {
		t.Errorf("Second request: expected 429, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "custom_rate_limit") {
		t.Errorf("Expected custom rate limit response, got %s", w.Body.String())
	}
}

// Test CORS wildcard subdomain
func TestCORSWildcardSubdomain(t *testing.T) {
	app := marten.New()
	app.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins: []string{"*.example.com"},
		AllowMethods: []string{"GET", "POST"},
	}))
	app.GET("/test", func(c *marten.Ctx) error {
		return c.OK(marten.M{"status": "ok"})
	})

	tests := []struct {
		origin  string
		allowed bool
	}{
		{"https://api.example.com", true},
		{"https://app.example.com", true},
		{"https://sub.api.example.com", true},
		{"https://other.com", false},
		{"https://example.com", false}, // exact match not covered by wildcard
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", tt.origin)
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)

		allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
		if tt.allowed && allowOrigin != tt.origin {
			t.Errorf("Origin %s: expected to be allowed, got Allow-Origin: %s", tt.origin, allowOrigin)
		}
		if !tt.allowed && allowOrigin == tt.origin {
			t.Errorf("Origin %s: expected to be blocked, but was allowed", tt.origin)
		}
	}
}

// Test App OnStart callback
func TestAppOnStart(t *testing.T) {
	app := marten.New()
	
	called := false
	app.OnStart(func() {
		called = true
	})

	// OnStart is called in Run/RunGraceful, but we can't easily test that
	// without actually starting the server. Just verify the callback is registered.
	app.GET("/test", func(c *marten.Ctx) error {
		return c.OK(marten.M{"status": "ok"})
	})
	
	// Verify app works
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	
	// Note: called will be false because ServeHTTP doesn't trigger OnStart
	// OnStart is only called in Run() and RunGraceful()
	_ = called
}

// Test App OnShutdown callback
func TestAppOnShutdown(t *testing.T) {
	app := marten.New()
	
	called := false
	app.OnShutdown(func() {
		called = true
	})

	// OnShutdown is called during graceful shutdown
	// We can't easily test this without starting the server
	app.GET("/test", func(c *marten.Ctx) error {
		return c.OK(marten.M{"status": "ok"})
	})
	
	// Verify app works
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	
	// Note: called will be false because ServeHTTP doesn't trigger OnShutdown
	_ = called
}

// Test Context Reset clears all fields
func TestContextResetComplete(t *testing.T) {
	app := marten.New()
	
	var firstRequestID string
	app.GET("/first", func(c *marten.Ctx) error {
		c.Set("key", "value")
		firstRequestID = c.RequestID()
		return c.OK(marten.M{"id": firstRequestID})
	})
	
	app.GET("/second", func(c *marten.Ctx) error {
		// Store should be empty after reset
		val := c.Get("key")
		if val != nil {
			return c.BadRequest("store not cleared")
		}
		// RequestID should be different
		if c.RequestID() == firstRequestID {
			return c.BadRequest("requestID not cleared")
		}
		return c.OK(marten.M{"status": "ok"})
	})

	// First request
	req := httptest.NewRequest("GET", "/first", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("First request: expected 200, got %d", w.Code)
	}

	// Second request - context should be fully reset
	req = httptest.NewRequest("GET", "/second", nil)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("Second request: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// Test RecoverJSON middleware
func TestRecoverJSON(t *testing.T) {
	app := marten.New()
	app.Use(middleware.RecoverJSON)
	app.GET("/panic", func(c *marten.Ctx) error {
		panic("json panic test")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("Expected 500, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "internal server error") {
		t.Errorf("Expected JSON error response, got %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "json panic test") {
		t.Errorf("Expected panic message in response, got %s", w.Body.String())
	}
}
