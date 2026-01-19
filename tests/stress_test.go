package tests

import (
	"bytes"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gomarten/marten"
	"github.com/gomarten/marten/middleware"
)

// --- Stress Tests for v0.1.3 ---

// Test high concurrency with context pooling
func TestHighConcurrencyContextPooling(t *testing.T) {
	app := marten.New()
	
	var counter int64
	app.GET("/increment", func(c *marten.Ctx) error {
		// Simulate some work
		val := atomic.AddInt64(&counter, 1)
		c.Set("value", val)
		time.Sleep(1 * time.Millisecond)
		
		// Verify context isolation
		stored := c.Get("value").(int64)
		if stored != val {
			return c.ServerError(fmt.Sprintf("context leak: expected %d, got %d", val, stored))
		}
		
		return c.JSON(200, marten.M{"count": val})
	})

	var wg sync.WaitGroup
	numRequests := 1000
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/increment", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Code != 200 {
				errors <- fmt.Errorf("request %d: expected 200, got %d: %s", id, rec.Code, rec.Body.String())
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}

	if atomic.LoadInt64(&counter) != int64(numRequests) {
		t.Errorf("expected counter=%d, got %d", numRequests, counter)
	}
}

// Test memory leaks with repeated requests
func TestMemoryLeakPrevention(t *testing.T) {
	app := marten.New()
	
	app.GET("/data", func(c *marten.Ctx) error {
		// Create some data
		data := make([]byte, 1024)
		for i := range data {
			data[i] = byte(i % 256)
		}
		c.Set("data", data)
		return c.Blob(200, "application/octet-stream", data)
	})

	// Run many requests to check for leaks
	for i := 0; i < 10000; i++ {
		req := httptest.NewRequest("GET", "/data", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != 200 {
			t.Fatalf("request %d failed: %d", i, rec.Code)
		}
	}
}

// Test route with many parameters
func TestManyParameters(t *testing.T) {
	app := marten.New()
	
	// Route with 10 parameters
	app.GET("/:p1/:p2/:p3/:p4/:p5/:p6/:p7/:p8/:p9/:p10", func(c *marten.Ctx) error {
		result := ""
		for i := 1; i <= 10; i++ {
			result += c.Param(fmt.Sprintf("p%d", i))
		}
		return c.Text(200, result)
	})

	req := httptest.NewRequest("GET", "/a/b/c/d/e/f/g/h/i/j", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "abcdefghij" {
		t.Errorf("expected 'abcdefghij', got %q", rec.Body.String())
	}
}

// Test deeply nested groups
func TestDeeplyNestedGroups(t *testing.T) {
	app := marten.New()
	
	g1 := app.Group("/api")
	g2 := g1.Group("/v1")
	g3 := g2.Group("/users")
	g4 := g3.Group("/:id")
	g5 := g4.Group("/posts")
	
	g5.GET("/:postId", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("id")+":"+c.Param("postId"))
	})

	req := httptest.NewRequest("GET", "/api/v1/users/123/posts/456", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "123:456" {
		t.Errorf("expected '123:456', got %q", rec.Body.String())
	}
}

// Test middleware chain with many middleware
func TestLongMiddlewareChain(t *testing.T) {
	app := marten.New()
	
	var order []int
	mu := sync.Mutex{}

	// Add 20 middleware
	for i := 0; i < 20; i++ {
		idx := i
		app.Use(func(next marten.Handler) marten.Handler {
			return func(c *marten.Ctx) error {
				mu.Lock()
				order = append(order, idx)
				mu.Unlock()
				return next(c)
			}
		})
	}

	app.GET("/test", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	mu.Lock()
	defer mu.Unlock()

	if len(order) != 20 {
		t.Errorf("expected 20 middleware calls, got %d", len(order))
	}

	// Verify order
	for i := 0; i < 20; i++ {
		if order[i] != i {
			t.Errorf("middleware order incorrect at position %d: expected %d, got %d", i, i, order[i])
		}
	}
}


// Test large request body handling
func TestLargeRequestBody(t *testing.T) {
	app := marten.New()
	
	app.POST("/upload", func(c *marten.Ctx) error {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return c.BadRequest(err.Error())
		}
		return c.JSON(200, marten.M{"size": len(body)})
	})

	// 10MB body
	largeBody := bytes.Repeat([]byte("x"), 10*1024*1024)
	req := httptest.NewRequest("POST", "/upload", bytes.NewReader(largeBody))
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

// Test large response body
func TestLargeResponseBody(t *testing.T) {
	app := marten.New()
	
	app.GET("/download", func(c *marten.Ctx) error {
		// 10MB response
		data := bytes.Repeat([]byte("x"), 10*1024*1024)
		return c.Blob(200, "application/octet-stream", data)
	})

	req := httptest.NewRequest("GET", "/download", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.Len() != 10*1024*1024 {
		t.Errorf("expected 10MB response, got %d bytes", rec.Body.Len())
	}
}

// Test route conflict detection with wildcards
func TestWildcardConflictDetection(t *testing.T) {
	app := marten.New()
	
	// This should work - wildcard and static at same level
	app.GET("/files/special", func(c *marten.Ctx) error {
		return c.Text(200, "special")
	})
	app.GET("/files/*path", func(c *marten.Ctx) error {
		return c.Text(200, "wildcard:"+c.Param("path"))
	})

	// Test static takes precedence
	req := httptest.NewRequest("GET", "/files/special", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "special" {
		t.Errorf("expected 'special', got %q", rec.Body.String())
	}

	// Test wildcard catches others
	req = httptest.NewRequest("GET", "/files/other.txt", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "wildcard:other.txt" {
		t.Errorf("expected 'wildcard:other.txt', got %q", rec.Body.String())
	}
}

// Test param and wildcard combination
func TestParamAndWildcardCombination(t *testing.T) {
	app := marten.New()
	
	app.GET("/users/:id/files/*path", func(c *marten.Ctx) error {
		return c.JSON(200, marten.M{
			"user": c.Param("id"),
			"path": c.Param("path"),
		})
	})

	tests := []struct {
		url      string
		user     string
		path     string
	}{
		{"/users/123/files/doc.pdf", "123", "doc.pdf"},
		{"/users/456/files/images/photo.png", "456", "images/photo.png"},
		{"/users/789/files/a/b/c/d.txt", "789", "a/b/c/d.txt"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.url, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != 200 {
			t.Errorf("url %s: expected 200, got %d", tt.url, rec.Code)
		}

		body := rec.Body.String()
		if !strings.Contains(body, tt.user) {
			t.Errorf("url %s: expected user %s in response", tt.url, tt.user)
		}
		if !strings.Contains(body, tt.path) {
			t.Errorf("url %s: expected path %s in response", tt.url, tt.path)
		}
	}
}

// Test special characters in params
func TestSpecialCharactersInParams(t *testing.T) {
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
		{"/files/file%20name", "file name"}, // URL decoded
		{"/files/file%2Bplus", "file+plus"},
		{"/files/100%25", "100%"},
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

// Test query string with special characters
func TestQueryStringSpecialCharacters(t *testing.T) {
	app := marten.New()
	
	app.GET("/search", func(c *marten.Ctx) error {
		return c.Text(200, c.Query("q"))
	})

	tests := []struct {
		query    string
		expected string
	}{
		{"?q=hello", "hello"},
		{"?q=hello%20world", "hello world"},
		{"?q=test%2Bplus", "test+plus"},
		{"?q=100%25", "100%"},
		{"?q=a%26b", "a&b"},
		{"?q=", ""},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/search"+tt.query, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != tt.expected {
			t.Errorf("query %s: expected %q, got %q", tt.query, tt.expected, rec.Body.String())
		}
	}
}

// Test middleware error propagation
func TestMiddlewareErrorPropagation(t *testing.T) {
	app := marten.New()
	
	var errorCaught error
	app.OnError(func(c *marten.Ctx, err error) {
		errorCaught = err
		c.JSON(500, marten.M{"error": err.Error()})
	})

	app.Use(func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			return fmt.Errorf("middleware error")
		}
	})

	app.GET("/test", func(c *marten.Ctx) error {
		return c.Text(200, "should not reach here")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if errorCaught == nil {
		t.Error("error should have been caught")
	}
	if errorCaught.Error() != "middleware error" {
		t.Errorf("expected 'middleware error', got %q", errorCaught.Error())
	}
}

// Test RateLimit with burst traffic
func TestRateLimitBurstTraffic(t *testing.T) {
	limiter := middleware.NewRateLimiter(middleware.RateLimitConfig{
		Requests: 10,
		Window:   time.Second,
	})
	defer limiter.Stop()

	app := marten.New()
	app.Use(limiter.Middleware())
	app.GET("/api", func(c *marten.Ctx) error {
		return c.OK(marten.M{"ok": true})
	})

	// Send 20 requests rapidly
	successCount := 0
	rateLimitedCount := 0

	for i := 0; i < 20; i++ {
		req := httptest.NewRequest("GET", "/api", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code == 200 {
			successCount++
		} else if rec.Code == 429 {
			rateLimitedCount++
		}
	}

	if successCount != 10 {
		t.Errorf("expected 10 successful requests, got %d", successCount)
	}
	if rateLimitedCount != 10 {
		t.Errorf("expected 10 rate limited requests, got %d", rateLimitedCount)
	}
}

// Test Timeout with context cancellation propagation
func TestTimeoutWithContextCancellation(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Timeout(50 * time.Millisecond))
	
	app.GET("/slow", func(c *marten.Ctx) error {
		select {
		case <-time.After(200 * time.Millisecond):
			return c.Text(200, "completed")
		case <-c.Context().Done():
			// Don't write response here - timeout middleware already wrote it
			return nil
		}
	})

	req := httptest.NewRequest("GET", "/slow", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should timeout
	if rec.Code != 504 {
		t.Errorf("expected 504 timeout, got %d", rec.Code)
	}
}
