// Middleware example - Using built-in and custom middleware
package main

import (
	"log"
	"time"

	"github.com/gomarten/marten"
	"github.com/gomarten/marten/middleware"
)

func main() {
	app := marten.New()

	// --- Built-in Middleware ---

	// Request ID - adds unique ID to each request
	app.Use(middleware.RequestID)

	// Logger - logs method, path, status, duration
	app.Use(middleware.Logger)

	// Recover - catches panics and returns 500
	app.Use(middleware.Recover)

	// Security headers
	app.Use(middleware.Secure(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		HSTSMaxAge:            31536000,
		HSTSIncludeSubdomains: true,
		ReferrerPolicy:        "strict-origin-when-cross-origin",
	}))

	// CORS with ExposeHeaders and MaxAge (v0.1.1)
	app.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins:     []string{"https://example.com", "https://app.example.com"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"X-Request-ID", "X-RateLimit-Remaining"},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}))

	// Body size limit (10MB)
	app.Use(middleware.BodyLimit(10 * middleware.MB))

	// Gzip compression
	app.Use(middleware.Compress(middleware.DefaultCompressConfig()))

	// --- Custom Middleware ---

	// Timing middleware
	app.Use(TimingMiddleware)

	// --- Routes ---

	app.GET("/", func(c *marten.Ctx) error {
		return c.OK(marten.M{
			"message":    "Hello with middleware!",
			"request_id": c.RequestID(),
		})
	})

	// Rate limited endpoint using NewRateLimiter (v0.1.1)
	// NewRateLimiter allows proper cleanup with Stop()
	rateLimiter := middleware.NewRateLimiter(middleware.RateLimitConfig{
		Requests: 10,
		Window:   time.Minute,
	})
	defer rateLimiter.Stop() // Clean up goroutine on shutdown

	limited := app.Group("/api")
	limited.Use(rateLimiter.Middleware())
	limited.GET("/limited", func(c *marten.Ctx) error {
		return c.OK(marten.M{"message": "This endpoint is rate limited"})
	})

	// Protected endpoint with basic auth
	admin := app.Group("/admin")
	admin.Use(middleware.BasicAuthSimple("admin", "secret123"))
	admin.GET("/dashboard", func(c *marten.Ctx) error {
		user := c.GetString("user")
		return c.OK(marten.M{
			"message": "Welcome to admin dashboard",
			"user":    user,
		})
	})

	// Timeout endpoint
	slow := app.Group("/slow")
	slow.Use(middleware.Timeout(2 * time.Second))
	slow.GET("/task", func(c *marten.Ctx) error {
		// Simulate slow operation
		select {
		case <-time.After(1 * time.Second):
			return c.OK(marten.M{"message": "Task completed"})
		case <-c.Context().Done():
			return c.ServerError("Task cancelled")
		}
	})

	// Cached endpoint with ETag
	cached := app.Group("/cached")
	cached.Use(middleware.ETag)
	cached.GET("/data", func(c *marten.Ctx) error {
		return c.OK(marten.M{
			"data":      "This response supports ETag caching",
			"timestamp": time.Now().Unix(),
		})
	})

	// No-cache endpoint
	nocache := app.Group("/nocache")
	nocache.Use(middleware.NoCache)
	nocache.GET("/data", func(c *marten.Ctx) error {
		return c.OK(marten.M{
			"data":      "This response is never cached",
			"timestamp": time.Now().Unix(),
		})
	})

	log.Println("Middleware example running on http://localhost:3000")
	app.Run(":3000")
}

// TimingMiddleware adds response time header
func TimingMiddleware(next marten.Handler) marten.Handler {
	return func(c *marten.Ctx) error {
		start := time.Now()
		err := next(c)
		c.Header("X-Response-Time", time.Since(start).String())
		return err
	}
}
