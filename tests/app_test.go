package tests

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/gomarten/marten"
)

func TestAppNew(t *testing.T) {
	app := marten.New()
	if app == nil {
		t.Fatal("New() returned nil")
	}
}

func TestAppServeHTTP(t *testing.T) {
	app := marten.New()
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

func TestAppOnError(t *testing.T) {
	app := marten.New()

	var capturedErr error
	app.OnError(func(c *marten.Ctx, err error) {
		capturedErr = err
		c.Text(500, "custom error")
	})

	app.GET("/", func(c *marten.Ctx) error {
		return errors.New("something went wrong")
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedErr == nil {
		t.Error("error handler was not called")
	}
	if capturedErr.Error() != "something went wrong" {
		t.Errorf("unexpected error: %v", capturedErr)
	}
	if rec.Body.String() != "custom error" {
		t.Errorf("expected 'custom error', got %q", rec.Body.String())
	}
}

func TestAppDefaultNotFound(t *testing.T) {
	app := marten.New()

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Errorf("expected 404, got %d", rec.Code)
	}
	if rec.Body.String() != "Not Found" {
		t.Errorf("expected 'Not Found', got %q", rec.Body.String())
	}
}

func TestAppCustomNotFound(t *testing.T) {
	app := marten.New()
	app.NotFound(func(c *marten.Ctx) error {
		return c.JSON(404, map[string]string{"error": "not found"})
	})

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

// --- Additional App Tests ---

func TestAppContextPooling(t *testing.T) {
	app := marten.New()

	var contexts []*marten.Ctx
	var mu sync.Mutex

	app.GET("/", func(c *marten.Ctx) error {
		mu.Lock()
		contexts = append(contexts, c)
		mu.Unlock()
		return c.Text(200, "ok")
	})

	// Make multiple requests
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	// Contexts should be reused (pooled)
	// We can't directly test this, but we verify requests work
	if len(contexts) != 10 {
		t.Errorf("expected 10 contexts, got %d", len(contexts))
	}
}

func TestAppMiddlewareOrder(t *testing.T) {
	app := marten.New()

	var order []string

	app.Use(func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			order = append(order, "global1-before")
			err := next(c)
			order = append(order, "global1-after")
			return err
		}
	})

	app.Use(func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			order = append(order, "global2-before")
			err := next(c)
			order = append(order, "global2-after")
			return err
		}
	})

	routeMw := func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			order = append(order, "route-before")
			err := next(c)
			order = append(order, "route-after")
			return err
		}
	}

	app.GET("/", func(c *marten.Ctx) error {
		order = append(order, "handler")
		return c.Text(200, "ok")
	}, routeMw)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	expected := []string{
		"global1-before", "global2-before", "route-before",
		"handler",
		"route-after", "global2-after", "global1-after",
	}

	if len(order) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(order), order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("position %d: expected %q, got %q", i, v, order[i])
		}
	}
}

func TestAppErrorHandlerNotCalledOnSuccess(t *testing.T) {
	app := marten.New()

	errorHandlerCalled := false
	app.OnError(func(c *marten.Ctx, err error) {
		errorHandlerCalled = true
	})

	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if errorHandlerCalled {
		t.Error("error handler should not be called on success")
	}
}

func TestAppConcurrentRequests(t *testing.T) {
	app := marten.New()
	app.GET("/users/:id", func(c *marten.Ctx) error {
		// Simulate some work
		id := c.Param("id")
		return c.JSON(200, map[string]string{"id": id})
	})

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/users/"+strconv.Itoa(id), nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Code != 200 {
				errors <- fmt.Errorf("request %d: expected 200, got %d", id, rec.Code)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}
