package middleware

import (
	"log"
	"time"

	"github.com/gomarten/marten"
)

// Logger logs request method, path, status code, and duration.
func Logger(next marten.Handler) marten.Handler {
	return func(c *marten.Ctx) error {
		start := time.Now()
		err := next(c)
		status := c.StatusCode()
		if status == 0 {
			status = 200
		}
		log.Printf("%s %s %d %v", c.Request.Method, c.Request.URL.Path, status, time.Since(start))
		return err
	}
}
