package middleware

import (
	"encoding/json"
	"fmt"
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
	// EnableColors enables colored output (default: false)
	EnableColors bool
	// JSONFormat outputs logs in JSON format (default: false)
	JSONFormat bool
}

// DefaultLoggerConfig returns sensible defaults.
func DefaultLoggerConfig() LoggerConfig {
	return LoggerConfig{
		Output: os.Stdout,
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

			method := c.Request.Method
			path := c.Request.URL.Path
			clientIP := c.ClientIP()

			// Custom format takes precedence
			if cfg.Format != nil {
				msg := cfg.Format(method, path, status, duration, clientIP)
				if msg != "" {
					logger.Println(msg)
				} else {
					logger.Printf("%s %s %d %v", method, path, status, duration)
				}
				return err
			}

			// JSON format
			if cfg.JSONFormat {
				logEntry := map[string]any{
					"time":      time.Now().UTC().Format(time.RFC3339),
					"method":    method,
					"path":      path,
					"status":    status,
					"duration":  duration.String(),
					"client_ip": clientIP,
				}
				if jsonBytes, jsonErr := json.Marshal(logEntry); jsonErr == nil {
					logger.Println(string(jsonBytes))
				}
				return err
			}

			// Colored format
			if cfg.EnableColors {
				logger.Println(formatWithColors(method, path, status, duration))
				return err
			}

			// Default format
			logger.Printf("%s %s %d %v", method, path, status, duration)
			return err
		}
	}
}

// formatWithColors returns a colored log line.
func formatWithColors(method, path string, status int, duration time.Duration) string {
	// ANSI color codes
	const (
		reset   = "\033[0m"
		red     = "\033[31m"
		green   = "\033[32m"
		yellow  = "\033[33m"
		blue    = "\033[34m"
		magenta = "\033[35m"
		cyan    = "\033[36m"
		white   = "\033[37m"
	)

	// Method color
	methodColor := cyan
	switch method {
	case "GET":
		methodColor = blue
	case "POST":
		methodColor = green
	case "PUT":
		methodColor = yellow
	case "DELETE":
		methodColor = red
	case "PATCH":
		methodColor = magenta
	}

	// Status color
	statusColor := green
	switch {
	case status >= 500:
		statusColor = red
	case status >= 400:
		statusColor = yellow
	case status >= 300:
		statusColor = cyan
	}

	return fmt.Sprintf("%s%-7s%s %s %s%d%s %v",
		methodColor, method, reset,
		path,
		statusColor, status, reset,
		duration,
	)
}
