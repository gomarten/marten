package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gomarten/marten"
	"github.com/gomarten/marten/middleware"
)

func TestGlobalMiddleware(t *testing.T) {
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
			err := next(c)
			order = append(order, "mw2-after")
			return err
		}
	})

	app.GET("/", func(c *marten.Ctx) error {
		order = append(order, "handler")
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(order), order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("position %d: expected %q, got %q", i, v, order[i])
		}
	}
}

func TestRouteSpecificMiddleware(t *testing.T) {
	app := marten.New()

	authMw := func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			if c.Request.Header.Get("Authorization") == "" {
				return c.Text(401, "unauthorized")
			}
			return next(c)
		}
	}

	app.GET("/public", func(c *marten.Ctx) error {
		return c.Text(200, "public")
	})

	app.GET("/private", func(c *marten.Ctx) error {
		return c.Text(200, "private")
	}, authMw)

	// Public route without auth
	req := httptest.NewRequest("GET", "/public", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("public: expected 200, got %d", rec.Code)
	}

	// Private route without auth
	req = httptest.NewRequest("GET", "/private", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Errorf("private without auth: expected 401, got %d", rec.Code)
	}

	// Private route with auth
	req = httptest.NewRequest("GET", "/private", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("private with auth: expected 200, got %d", rec.Code)
	}
}

func TestMiddlewareChain(t *testing.T) {
	var order []string

	mw1 := func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			order = append(order, "1")
			return next(c)
		}
	}

	mw2 := func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			order = append(order, "2")
			return next(c)
		}
	}

	mw3 := func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			order = append(order, "3")
			return next(c)
		}
	}

	chained := marten.Chain(mw1, mw2, mw3)
	handler := chained(func(c *marten.Ctx) error {
		order = append(order, "handler")
		return nil
	})

	handler(&marten.Ctx{})

	expected := []string{"1", "2", "3", "handler"}
	if len(order) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("position %d: expected %q, got %q", i, v, order[i])
		}
	}
}

// --- CORS Middleware Tests ---

func TestCORSMiddleware(t *testing.T) {
	app := marten.New()
	app.Use(middleware.CORS(middleware.DefaultCORSConfig()))
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected wildcard origin, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSSpecificOrigin(t *testing.T) {
	app := marten.New()
	app.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins: []string{"https://allowed.com"},
		AllowMethods: []string{"GET", "POST"},
		AllowHeaders: []string{"Content-Type"},
	}))
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	// Allowed origin
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://allowed.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "https://allowed.com" {
		t.Errorf("expected allowed origin, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}

	// Disallowed origin
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("expected no CORS header for disallowed origin, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSPreflight(t *testing.T) {
	app := marten.New()
	app.Use(middleware.CORS(middleware.DefaultCORSConfig()))
	app.POST("/api", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("OPTIONS", "/api", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 204 {
		t.Errorf("expected 204 for preflight, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected Allow-Methods header")
	}
}

func TestCORSCredentials(t *testing.T) {
	app := marten.New()
	app.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins:     []string{"https://trusted.com"},
		AllowMethods:     []string{"GET"},
		AllowCredentials: true,
	}))
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://trusted.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("expected credentials header")
	}
}

func TestCORSWildcardNoCredentials(t *testing.T) {
	// Credentials should not be set with wildcard origin (security)
	app := marten.New()
	app.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET"},
		AllowCredentials: true, // Should be ignored with wildcard
	}))
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Credentials") == "true" {
		t.Error("credentials should not be allowed with wildcard origin")
	}
}

// --- Logger Middleware Tests ---

func TestLoggerMiddleware(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Logger)
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

// --- Recover Middleware Tests ---

func TestRecoverMiddleware(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Recover)
	app.GET("/panic", func(c *marten.Ctx) error {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 500 {
		t.Errorf("expected 500 after panic, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Internal Server Error") {
		t.Errorf("expected error message, got %q", rec.Body.String())
	}
}

func TestRecoverNoPanic(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Recover)
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

// --- RequestID Middleware Tests ---

func TestRequestIDMiddleware(t *testing.T) {
	app := marten.New()
	app.Use(middleware.RequestID)
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, c.RequestID())
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	id := rec.Header().Get("X-Request-ID")
	if id == "" {
		t.Error("expected X-Request-ID header")
	}
	if rec.Body.String() != id {
		t.Errorf("body should match header: %q != %q", rec.Body.String(), id)
	}
}

func TestRequestIDPreserved(t *testing.T) {
	app := marten.New()
	app.Use(middleware.RequestID)
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, c.RequestID())
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "custom-123")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") != "custom-123" {
		t.Errorf("expected preserved ID, got %q", rec.Header().Get("X-Request-ID"))
	}
}

// --- BasicAuth Middleware Tests ---

func TestBasicAuthSimple(t *testing.T) {
	app := marten.New()
	app.Use(middleware.BasicAuthSimple("admin", "secret"))
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "welcome "+c.GetString("user"))
	})

	// No auth
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Errorf("expected 401 without auth, got %d", rec.Code)
	}
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Error("expected WWW-Authenticate header")
	}

	// Wrong credentials
	req = httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("admin", "wrong")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Errorf("expected 401 with wrong password, got %d", rec.Code)
	}

	// Correct credentials
	req = httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("admin", "secret")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("expected 200 with correct auth, got %d", rec.Code)
	}
	if rec.Body.String() != "welcome admin" {
		t.Errorf("expected 'welcome admin', got %q", rec.Body.String())
	}
}

func TestBasicAuthCustomValidator(t *testing.T) {
	users := map[string]string{
		"alice": "password1",
		"bob":   "password2",
	}

	app := marten.New()
	app.Use(middleware.BasicAuth(middleware.BasicAuthConfig{
		Realm: "Test Realm",
		Validate: func(user, pass string) bool {
			expected, ok := users[user]
			return ok && expected == pass
		},
	}))
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "hello "+c.GetString("user"))
	})

	// Alice
	req := httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("alice", "password1")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Body.String() != "hello alice" {
		t.Errorf("expected 'hello alice', got %q", rec.Body.String())
	}

	// Bob
	req = httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("bob", "password2")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Body.String() != "hello bob" {
		t.Errorf("expected 'hello bob', got %q", rec.Body.String())
	}
}

// --- RateLimit Middleware Tests ---

func TestRateLimitMiddleware(t *testing.T) {
	app := marten.New()
	app.Use(middleware.RateLimit(middleware.RateLimitConfig{
		Requests: 3,
		Window:   time.Minute,
		KeyFunc:  func(c *marten.Ctx) string { return "test" }, // Same key for all
	}))
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	// First 3 requests should succeed
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Errorf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// 4th request should be rate limited
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 429 {
		t.Errorf("expected 429 after rate limit, got %d", rec.Code)
	}
}

func TestRateLimitByIP(t *testing.T) {
	app := marten.New()
	app.Use(middleware.RateLimit(middleware.RateLimitConfig{
		Requests: 2,
		Window:   time.Minute,
	}))
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	// Different IPs should have separate limits
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Forwarded-For", "1.1.1.1")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Errorf("IP1 request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	// IP1 should be limited
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "1.1.1.1")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 429 {
		t.Errorf("IP1 should be limited, got %d", rec.Code)
	}

	// IP2 should still work
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "2.2.2.2")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("IP2 should not be limited, got %d", rec.Code)
	}
}

func TestRateLimitConcurrent(t *testing.T) {
	app := marten.New()
	app.Use(middleware.RateLimit(middleware.RateLimitConfig{
		Requests: 10,
		Window:   time.Minute,
		KeyFunc:  func(c *marten.Ctx) string { return "concurrent" },
	}))
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	var wg sync.WaitGroup
	results := make(chan int, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
			results <- rec.Code
		}()
	}

	wg.Wait()
	close(results)

	successCount := 0
	limitedCount := 0
	for code := range results {
		if code == 200 {
			successCount++
		} else if code == 429 {
			limitedCount++
		}
	}

	if successCount != 10 {
		t.Errorf("expected 10 successes, got %d", successCount)
	}
	if limitedCount != 10 {
		t.Errorf("expected 10 rate limited, got %d", limitedCount)
	}
}

// --- Timeout Middleware Tests ---

func TestTimeoutMiddleware(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Timeout(50 * time.Millisecond))
	app.GET("/slow", func(c *marten.Ctx) error {
		time.Sleep(200 * time.Millisecond)
		return c.Text(200, "done")
	})
	app.GET("/fast", func(c *marten.Ctx) error {
		return c.Text(200, "fast")
	})

	// Fast request should succeed
	req := httptest.NewRequest("GET", "/fast", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("fast: expected 200, got %d", rec.Code)
	}

	// Slow request should timeout
	req = httptest.NewRequest("GET", "/slow", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("slow: expected 504, got %d", rec.Code)
	}
}

// --- Middleware Combination Tests ---

func TestMiddlewareStack(t *testing.T) {
	app := marten.New()
	app.Use(
		middleware.RequestID,
		middleware.Logger,
		middleware.Recover,
	)
	app.GET("/", func(c *marten.Ctx) error {
		return c.OK(marten.M{"request_id": c.RequestID()})
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header")
	}
}
