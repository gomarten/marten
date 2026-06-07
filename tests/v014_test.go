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

// ─── Bug fix: Routes() root path ───────────────────────────────────────────

func TestRoutesRootPath(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error { return nil })
	app.GET("/users", func(c *marten.Ctx) error { return nil })
	app.POST("/users", func(c *marten.Ctx) error { return nil })
	app.GET("/users/:id", func(c *marten.Ctx) error { return nil })
	app.GET("/files/*path", func(c *marten.Ctx) error { return nil })

	routes := app.Routes()

	found := make(map[string]bool)
	for _, r := range routes {
		found[r.Method+":"+r.Path] = true
	}

	if !found["GET:/"] {
		t.Error("root route should be '/', got empty string")
	}
	if !found["GET:/users"] {
		t.Error("missing GET /users")
	}
	if !found["POST:/users"] {
		t.Error("missing POST /users")
	}
	if !found["GET:/users/:id"] {
		t.Error("missing GET /users/:id")
	}
	if !found["GET:/files/*path"] {
		t.Error("missing GET /files/*path")
	}
}

func TestRoutesSorted(t *testing.T) {
	app := marten.New()
	app.GET("/z", func(c *marten.Ctx) error { return nil })
	app.GET("/a", func(c *marten.Ctx) error { return nil })
	app.POST("/a", func(c *marten.Ctx) error { return nil })
	app.GET("/m", func(c *marten.Ctx) error { return nil })

	routes := app.Routes()

	for i := 1; i < len(routes); i++ {
		prev := routes[i-1]
		curr := routes[i]
		if curr.Path < prev.Path {
			t.Errorf("routes not sorted by path: %q before %q", prev.Path, curr.Path)
		}
		if curr.Path == prev.Path && curr.Method < prev.Method {
			t.Errorf("routes with same path not sorted by method: %q before %q", prev.Method, curr.Method)
		}
	}
}

// ─── Bug fix: Timeout data race ────────────────────────────────────────────

func TestTimeoutNoRace(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Timeout(30 * time.Millisecond))
	app.GET("/slow", func(c *marten.Ctx) error {
		time.Sleep(120 * time.Millisecond)
		// This write must be a no-op; should not race with the 504 response.
		return c.Text(200, "done")
	})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/slow", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
			if rec.Code != http.StatusGatewayTimeout {
				t.Errorf("expected 504, got %d", rec.Code)
			}
		}()
	}
	wg.Wait()
}

func TestTimeoutWithConfigNoRace(t *testing.T) {
	app := marten.New()
	app.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: 30 * time.Millisecond,
		OnTimeout: func(c *marten.Ctx) error {
			return c.JSON(http.StatusServiceUnavailable, marten.E("custom timeout"))
		},
	}))
	app.GET("/slow", func(c *marten.Ctx) error {
		time.Sleep(120 * time.Millisecond)
		return c.Text(200, "done")
	})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/slow", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
			if rec.Code != http.StatusServiceUnavailable {
				t.Errorf("expected 503, got %d", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), "custom timeout") {
				t.Errorf("expected custom timeout message, got %s", rec.Body.String())
			}
		}()
	}
	wg.Wait()
}

// ─── Bug fix: BodyLimit 413 on chunked read ────────────────────────────────

func TestBodyLimitChunked413(t *testing.T) {
	app := marten.New()
	app.Use(middleware.BodyLimit(10)) // 10 bytes
	app.POST("/upload", func(c *marten.Ctx) error {
		// Reading the body triggers the limit check.
		buf := make([]byte, 100)
		n, err := c.Request.Body.Read(buf)
		if err != nil {
			return err // BodyLimit middleware should catch this
		}
		return c.Text(200, string(buf[:n]))
	})

	// Body over limit, no Content-Length (chunked style)
	req := httptest.NewRequest("POST", "/upload", strings.NewReader("12345678901"))
	req.ContentLength = -1
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413 for chunked over-limit body, got %d", rec.Code)
	}
}

// ─── New feature: QueryFloat64 ─────────────────────────────────────────────

func TestQueryFloat64(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		v := c.QueryFloat64("amount")
		return c.JSON(200, marten.M{"amount": v})
	})

	tests := []struct {
		query    string
		expected float64
	}{
		{"?amount=3.14", 3.14},
		{"?amount=0", 0},
		{"?amount=-1.5", -1.5},
		{"?amount=1e3", 1000},
		{"?amount=invalid", 0},
		{"", 0},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/"+tt.query, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != 200 {
			t.Errorf("query %q: expected 200, got %d", tt.query, rec.Code)
		}
	}
}

// ─── New feature: GetFloat64 ───────────────────────────────────────────────

func TestGetFloat64(t *testing.T) {
	app := marten.New()

	app.Use(func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			c.Set("score", float64(9.99))
			c.Set("invalid", "not a float")
			return next(c)
		}
	})

	app.GET("/", func(c *marten.Ctx) error {
		score := c.GetFloat64("score")
		invalid := c.GetFloat64("invalid")
		missing := c.GetFloat64("missing")

		return c.JSON(200, marten.M{
			"score":   score,
			"invalid": invalid,
			"missing": missing,
		})
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "9.99") {
		t.Errorf("expected score=9.99, got %s", body)
	}
	if !strings.Contains(body, `"invalid":0`) {
		t.Errorf("expected invalid=0, got %s", body)
	}
	if !strings.Contains(body, `"missing":0`) {
		t.Errorf("expected missing=0, got %s", body)
	}
}

// ─── New feature: c.Accepted ───────────────────────────────────────────────

func TestContextAccepted(t *testing.T) {
	app := marten.New()
	app.POST("/jobs", func(c *marten.Ctx) error {
		return c.Accepted(marten.M{"job_id": "abc-123", "status": "queued"})
	})

	req := httptest.NewRequest("POST", "/jobs", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("expected JSON content type, got %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "abc-123") {
		t.Errorf("expected job_id in response, got %s", rec.Body.String())
	}
}

// ─── New feature: middleware.Health ────────────────────────────────────────

func TestHealthMiddleware(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Health("/health"))
	app.GET("/api/data", func(c *marten.Ctx) error {
		return c.Text(200, "data")
	})

	// Health endpoint should respond 200 {"status":"ok"}
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("health: expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status"`) {
		t.Errorf("health: expected JSON body, got %q", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Errorf("health: expected status ok, got %q", rec.Body.String())
	}

	// Non-health path should pass through
	req = httptest.NewRequest("GET", "/api/data", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("api/data: expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "data" {
		t.Errorf("api/data: expected 'data', got %q", rec.Body.String())
	}
}

func TestHealthMiddlewarePOSTIgnored(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Health("/health"))
	app.POST("/health", func(c *marten.Ctx) error {
		return c.Text(200, "post handler")
	})

	// POST /health should not be intercepted by the health middleware
	req := httptest.NewRequest("POST", "/health", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "post handler" {
		t.Errorf("expected POST to reach handler, got %q", rec.Body.String())
	}
}

func TestHealthWithConfig(t *testing.T) {
	app := marten.New()
	app.Use(middleware.HealthWithConfig(middleware.HealthConfig{
		Path: "/healthz",
		Handler: func(c *marten.Ctx) error {
			return c.OK(marten.M{
				"status":  "ok",
				"version": "1.0.0",
			})
		},
	}))

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "1.0.0") {
		t.Errorf("expected version in response, got %q", rec.Body.String())
	}
}

func TestHealthDefaultPath(t *testing.T) {
	// HealthWithConfig with empty path should default to /health
	app := marten.New()
	app.Use(middleware.HealthWithConfig(middleware.HealthConfig{}))

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200 at default /health, got %d", rec.Code)
	}
}

func TestHealthDoesNotBlockLogger(t *testing.T) {
	// Health middleware should work before or after logger without issues
	app := marten.New()
	app.Use(middleware.Health("/ping"))
	app.Use(middleware.Logger)

	req := httptest.NewRequest("GET", "/ping", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

// ─── Bug fix: Routes() path building with groups ───────────────────────────

func TestRoutesWithGroups(t *testing.T) {
	app := marten.New()

	api := app.Group("/api/v1")
	api.GET("/users", func(c *marten.Ctx) error { return nil })
	api.POST("/users", func(c *marten.Ctx) error { return nil })
	api.GET("/users/:id", func(c *marten.Ctx) error { return nil })

	app.GET("/", func(c *marten.Ctx) error { return nil })

	routes := app.Routes()
	found := make(map[string]bool)
	for _, r := range routes {
		found[r.Method+":"+r.Path] = true
	}

	if !found["GET:/"] {
		t.Error("missing GET /")
	}
	if !found["GET:/api/v1/users"] {
		t.Error("missing GET /api/v1/users")
	}
	if !found["POST:/api/v1/users"] {
		t.Error("missing POST /api/v1/users")
	}
	if !found["GET:/api/v1/users/:id"] {
		t.Error("missing GET /api/v1/users/:id")
	}
}

// ─── Timeout: fast handler still works after fix ───────────────────────────

func TestTimeoutFastHandlerUnaffected(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Timeout(200 * time.Millisecond))
	app.GET("/fast", func(c *marten.Ctx) error {
		return c.Text(200, "quick")
	})

	req := httptest.NewRequest("GET", "/fast", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("fast handler: expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "quick" {
		t.Errorf("fast handler: expected 'quick', got %q", rec.Body.String())
	}
}

// ─── Timeout: context cancellation is propagated ───────────────────────────

func TestTimeoutContextPropagated(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Timeout(50 * time.Millisecond))
	app.GET("/ctx", func(c *marten.Ctx) error {
		select {
		case <-time.After(200 * time.Millisecond):
			return c.Text(200, "late")
		case <-c.Context().Done():
			// Context was cancelled by timeout — return without writing.
			return nil
		}
	})

	req := httptest.NewRequest("GET", "/ctx", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("expected 504, got %d", rec.Code)
	}
}

// ─── NewCtx public API ─────────────────────────────────────────────────────

func TestNewCtxPublicAPI(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	c := marten.NewCtx(rec, req)
	if err := c.Text(200, "hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "hello" {
		t.Errorf("expected 'hello', got %q", rec.Body.String())
	}
}
