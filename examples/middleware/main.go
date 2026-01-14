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

	// --- Lifecycle Hooks (v0.1.2) ---
	app.OnStart(func() {
		log.Println("Server starting up...")
	})
	app.OnShutdown(func() {
		log.Println("Server shutting down, cleaning up...")
	})

	// --- Built-in Middleware ---

	// Request ID - adds unique ID to each request
	app.Use(middleware.RequestID)

	// Logger with colored output (v0.1.2)
	app.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		EnableColors: true,
	}))

	// Recover with JSON response (v0.1.2)
	app.Use(middleware.RecoverJSON)

	// Security headers
	app.Use(middleware.Secure(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		HSTSMaxAge:            31536000,
		HSTSIncludeSubdomains: true,
		ReferrerPolicy:        "strict-origin-when-cross-origin",
	}))

	// CORS with wildcard subdomain support (v0.1.2)
	app.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins:     []string{"*.example.com", "http://localhost:3000"},
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

	// Rate limited endpoint with custom response (v0.1.2)
	rateLimiter := middleware.NewRateLimiter(middleware.RateLimitConfig{
		Requests: 10,
		Window:   time.Minute,
		OnLimitReached: func(c *marten.Ctx) error {
			return c.JSON(429, marten.M{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests, please slow down",
			})
		},
	})
	defer rateLimiter.Stop() // Clean up goroutine on shutdown

	limited := app.Group("/api")
	limited.Use(rateLimiter.Middleware())
	limited.GET("/limited", func(c *marten.Ctx) error {
		return c.OK(marten.M{"message": "This endpoint is rate limited"})
	})

	// Form binding endpoint (v0.1.2 - supports form-urlencoded)
	app.POST("/form", func(c *marten.Ctx) error {
		var data struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if err := c.Bind(&data); err != nil {
			return c.BadRequest(err.Error())
		}
		return c.OK(marten.M{"received": data})
	})

	// Protected endpoint with basic auth
	admin := app.Group("/admin")
	admin.Use(middleware.BasicAuth(middleware.BasicAuthConfig{
		Realm: "Admin Area",
		Validate: func(user, pass string) bool {
			return user == "admin" && pass == "secret123"
		},
	}))
	admin.GET("/dashboard", func(c *marten.Ctx) error {
		user := c.GetString("user")
		return c.OK(marten.M{
			"message": "Welcome to admin dashboard",
			"user":    user,
		})
	})

	// Timeout endpoint with custom response (v0.1.2)
	slow := app.Group("/slow")
	slow.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: 2 * time.Second,
		OnTimeout: func(c *marten.Ctx) error {
			return c.JSON(504, marten.M{
				"error":   "timeout",
				"message": "Request took too long",
			})
		},
	}))
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

	// Panic endpoint to test RecoverJSON
	app.GET("/panic", func(c *marten.Ctx) error {
		panic("intentional panic for testing")
	})

	log.Println("Middleware example running on http://localhost:3000")
	log.Println("Try: GET /, POST /form, GET /api/limited, GET /panic")
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
