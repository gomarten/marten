package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gomarten/marten"
)

// Timeout returns a middleware that times out requests.
// The handler will be cancelled when the timeout is reached.
func Timeout(d time.Duration) marten.Middleware {
	return func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			ctx, cancel := context.WithTimeout(c.Request.Context(), d)
			defer cancel()

			c.Request = c.Request.WithContext(ctx)

			done := make(chan error, 1)

			go func() {
				defer func() {
					if r := recover(); r != nil {
						done <- c.JSON(http.StatusInternalServerError, marten.E("internal error"))
					}
				}()
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

// TimeoutWithConfig returns a timeout middleware with custom error handler.
type TimeoutConfig struct {
	Timeout time.Duration
	OnTimeout func(c *marten.Ctx) error
}

// TimeoutWithConfig returns a timeout middleware with configuration.
func TimeoutWithConfig(cfg TimeoutConfig) marten.Middleware {
	if cfg.OnTimeout == nil {
		cfg.OnTimeout = func(c *marten.Ctx) error {
			return c.JSON(http.StatusGatewayTimeout, marten.E("request timeout"))
		}
	}

	return func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			ctx, cancel := context.WithTimeout(c.Request.Context(), cfg.Timeout)
			defer cancel()

			c.Request = c.Request.WithContext(ctx)

			var (
				handlerErr error
				once       sync.Once
				done       = make(chan struct{})
			)

			go func() {
				defer close(done)
				defer func() {
					if r := recover(); r != nil {
						once.Do(func() {
							handlerErr = c.JSON(http.StatusInternalServerError, marten.E("internal error"))
						})
					}
				}()
				once.Do(func() {
					handlerErr = next(c)
				})
			}()

			select {
			case <-done:
				return handlerErr
			case <-ctx.Done():
				return cfg.OnTimeout(c)
			}
		}
	}
}
