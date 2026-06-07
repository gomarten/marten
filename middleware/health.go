package middleware

import (
	"net/http"

	"github.com/gomarten/marten"
)

// HealthConfig configures the health check endpoint.
type HealthConfig struct {
	// Path is the URL path for the health check (default: "/health").
	Path string
	// Handler is a custom handler for the health check response.
	// If nil, responds with 200 {"status":"ok"}.
	Handler marten.Handler
}

// Health returns a middleware that intercepts requests to a health-check
// path and responds immediately, bypassing all downstream handlers and
// middleware registered after it.
//
// Usage:
//
//	app.Use(middleware.Health("/health"))
//
// Or with a custom response:
//
//	app.Use(middleware.HealthWithConfig(middleware.HealthConfig{
//	    Path: "/healthz",
//	    Handler: func(c *marten.Ctx) error {
//	        return c.OK(marten.M{"status": "ok", "version": "1.0.0"})
//	    },
//	}))
func Health(path string) marten.Middleware {
	return HealthWithConfig(HealthConfig{Path: path})
}

// HealthWithConfig returns a health check middleware with custom configuration.
func HealthWithConfig(cfg HealthConfig) marten.Middleware {
	if cfg.Path == "" {
		cfg.Path = "/health"
	}
	if cfg.Handler == nil {
		cfg.Handler = func(c *marten.Ctx) error {
			return c.JSON(http.StatusOK, marten.M{"status": "ok"})
		}
	}

	return func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			if c.Request.Method == http.MethodGet && c.Request.URL.Path == cfg.Path {
				return cfg.Handler(c)
			}
			return next(c)
		}
	}
}
