package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gomarten/marten"
)

// Timeout returns a middleware that times out requests.
func Timeout(d time.Duration) marten.Middleware {
	return func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			ctx, cancel := context.WithTimeout(c.Request.Context(), d)
			defer cancel()

			c.Request = c.Request.WithContext(ctx)

			done := make(chan error, 1)
			go func() {
				done <- next(c)
			}()

			select {
			case err := <-done:
				return err
			case <-ctx.Done():
				return c.JSON(http.StatusGatewayTimeout, marten.E("request timeout"))
			}
		}
	}
}
