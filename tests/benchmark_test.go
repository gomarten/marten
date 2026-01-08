package tests

import (
	"net/http/httptest"
	"testing"

	"github.com/gomarten/marten"
	"github.com/gomarten/marten/middleware"
)

func BenchmarkRouterStatic(b *testing.B) {
	app := marten.New()
	app.GET("/users", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/users", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

func BenchmarkRouterParam(b *testing.B) {
	app := marten.New()
	app.GET("/users/:id", func(c *marten.Ctx) error {
		_ = c.Param("id")
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/users/123", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

func BenchmarkRouterMultipleParams(b *testing.B) {
	app := marten.New()
	app.GET("/users/:id/posts/:postId", func(c *marten.Ctx) error {
		_ = c.Param("id")
		_ = c.Param("postId")
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/users/123/posts/456", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

func BenchmarkJSON(b *testing.B) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		return c.JSON(200, map[string]string{"message": "hello"})
	})

	req := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

func BenchmarkMiddlewareStack(b *testing.B) {
	app := marten.New()
	app.Use(middleware.Logger, middleware.Recover)
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

func BenchmarkContextHelpers(b *testing.B) {
	app := marten.New()
	app.GET("/search", func(c *marten.Ctx) error {
		_ = c.Query("q")
		_ = c.QueryInt("page")
		_ = c.QueryDefault("sort", "created_at")
		_ = c.ClientIP()
		_ = c.RequestID()
		return c.OK(marten.M{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/search?q=test&page=5", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

func BenchmarkParallel(b *testing.B) {
	app := marten.New()
	app.GET("/users/:id", func(c *marten.Ctx) error {
		return c.OK(marten.M{"id": c.Param("id")})
	})

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/users/123", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
		}
	})
}
