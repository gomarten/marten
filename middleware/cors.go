package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gomarten/marten"
)

// CORSConfig holds CORS configuration.
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string // Headers that can be exposed to the client
	AllowCredentials bool
	MaxAge           int // Preflight cache duration in seconds
}

// DefaultCORSConfig returns a permissive CORS config.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// CORS returns a CORS middleware with the given config.
func CORS(cfg CORSConfig) marten.Middleware {
	hasWildcard := false
	for _, o := range cfg.AllowOrigins {
		if o == "*" {
			hasWildcard = true
			break
		}
	}

	return func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			origin := c.Request.Header.Get("Origin")

			// Set Vary header for proper caching
			c.Header("Vary", "Origin")

			allowed := false
			for _, o := range cfg.AllowOrigins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}

			if allowed {
				if hasWildcard {
					c.Header("Access-Control-Allow-Origin", "*")
				} else {
					c.Header("Access-Control-Allow-Origin", origin)
				}
				c.Header("Access-Control-Allow-Methods", strings.Join(cfg.AllowMethods, ", "))
				c.Header("Access-Control-Allow-Headers", strings.Join(cfg.AllowHeaders, ", "))

				if len(cfg.ExposeHeaders) > 0 {
					c.Header("Access-Control-Expose-Headers", strings.Join(cfg.ExposeHeaders, ", "))
				}

				if cfg.AllowCredentials && !hasWildcard {
					c.Header("Access-Control-Allow-Credentials", "true")
				}

				if cfg.MaxAge > 0 {
					c.Header("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
				}
			}

			if c.Request.Method == http.MethodOptions {
				c.Status(http.StatusNoContent)
				return nil
			}

			return next(c)
		}
	}
}
