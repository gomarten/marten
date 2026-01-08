package middleware

import "github.com/gomarten/marten"

// RequestID adds a unique request ID to each request.
func RequestID(next marten.Handler) marten.Handler {
	return func(c *marten.Ctx) error {
		id := c.RequestID()
		c.Header("X-Request-ID", id)
		return next(c)
	}
}
