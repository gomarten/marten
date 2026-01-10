package middleware

import (
	"io"
	"log"
	"os"
	"time"

	"github.com/gomarten/marten"
)

// LoggerConfig configures the logger middleware.
type LoggerConfig struct {
	// Output is the writer for log output (default: os.Stdout)
	Output io.Writer
	// Format is a function that formats the log message
	Format func(method, path string, status int, duration time.Duration, clientIP string) string
	// Skip is a function to skip logging for certain requests
	Skip func(*marten.Ctx) bool
}

// DefaultLoggerConfig returns sensible defaults.
func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Output: os.Stdout,
		Format: func(method, path string, status int, duration time.Duration, clientIP string) string {
			return ""
		},
	}
}

// Logger logs request method, path, status code, and duration.
func Logger(next marten.Handler) marten.Handler {
	return LoggerWithConfig(DefaultLoggerConfig())(next)
}

// LoggerWithConfig returns a logger middleware with custom config.
func LoggerWithConfig(cfg LoggerConfig) marten.Middleware {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}
	logger := log.New(cfg.Output, "", 0)

	return func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			if cfg.Skip != nil && cfg.Skip(c) {
				return next(c)
			}

			start := time.Now()
			err := next(c)
			duration := time.Since(start)

			status := c.StatusCode()
			if status == 0 {
				status = 200
			}

			if cfg.Format != nil {
				msg := cfg.Format(c.Request.Method, c.Request.URL.Path, status, duration, c.ClientIP())
				if msg != "" {
					logger.Println(msg)
				} else {
					logger.Printf("%s %s %d %v", c.Request.Method, c.Request.URL.Path, status, duration)
				}
			} else {
				logger.Printf("%s %s %d %v", c.Request.Method, c.Request.URL.Path, status, duration)
			}

			return err
		}
	}
}
