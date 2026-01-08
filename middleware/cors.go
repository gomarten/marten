package middleware

import (
	"net/http"
	"strings"

	"github.com/gomarten/marten"
)

// CORSConfig holds CORS configuration.
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
}

// DefaultCORSConfig returns a permissive CORS config.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: false,
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

				if cfg.AllowCredentials && !hasWildcard {
					c.Header("Access-Control-Allow-Credentials", "true")
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
