package middleware

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gomarten/marten"
)

// Timeout returns a middleware that times out requests after d.
// Uses TimeoutWithConfig internally.
func Timeout(d time.Duration) marten.Middleware {
	return TimeoutWithConfig(TimeoutConfig{Timeout: d})
}

// TimeoutConfig configures the timeout middleware.
type TimeoutConfig struct {
	// Timeout is the maximum duration allowed for a handler to complete.
	Timeout time.Duration
	// OnTimeout is called when the timeout fires. If nil, responds with
	// 504 {"error":"request timeout"}.
	OnTimeout func(c *marten.Ctx) error
}

// TimeoutWithConfig returns a timeout middleware with custom configuration.
//
// The handler runs in its own goroutine. c.Writer is replaced with a
// timeoutWriter that drops all writes once claim() is called by the timeout
// path. After claiming, the middleware drains the done channel — waiting for
// the handler goroutine to exit — before writing the timeout response to the
// original ResponseWriter. This guarantees there is never concurrent access
// to the underlying ResponseWriter from two goroutines.
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

			// tw intercepts the handler goroutine's ResponseWriter access.
			// claim() makes all further Write/WriteHeader calls no-ops so
			// the goroutine can safely finish without touching the wire.
			origWriter := c.Writer
			tw := &timeoutWriter{ResponseWriter: origWriter}
			c.Writer = tw

			type result struct{ err error }
			done := make(chan result, 1)

			go func() {
				err := func() (e error) {
					defer func() {
						if r := recover(); r != nil {
							e = c.JSON(http.StatusInternalServerError, marten.E("internal error"))
						}
					}()
					return next(c)
				}()
				done <- result{err}
			}()

			select {
			case res := <-done:
				// Handler finished before the deadline — restore writer and return.
				c.Writer = origWriter
				return res.err

			case <-ctx.Done():
				// Drop the handler goroutine's write slot so any in-flight or
				// future write through tw becomes a no-op.
				tw.claim()

				// IMPORTANT: wait for the goroutine to finish BEFORE writing
				// to origWriter. This is the only way to guarantee zero
				// concurrent access to the underlying ResponseWriter.
				<-done

				// Now we own origWriter exclusively — write the timeout response.
				// Reset c.written so the timeout handler can write freely even
				// if the handler goroutine already set it (those writes were
				// dropped by tw and never reached origWriter).
				c.Writer = origWriter
				c.ResetWritten()
				return cfg.OnTimeout(c)
			}
		}
	}
}

// timeoutWriter wraps http.ResponseWriter. Once claim() is called every
// subsequent Write and WriteHeader is silently discarded, eliminating
// data races between a slow handler goroutine and the timeout path.
type timeoutWriter struct {
	http.ResponseWriter
	claimed int32 // accessed atomically; 1 = timeout path has won
}

// claim marks this writer as owned by the timeout path.
// Returns true on the first call, false on every subsequent call.
func (tw *timeoutWriter) claim() bool {
	return atomic.CompareAndSwapInt32(&tw.claimed, 0, 1)
}

func (tw *timeoutWriter) isClaimed() bool {
	return atomic.LoadInt32(&tw.claimed) != 0
}

// WriteHeader is a no-op after claim() has been called.
func (tw *timeoutWriter) WriteHeader(code int) {
	if tw.isClaimed() {
		return
	}
	tw.ResponseWriter.WriteHeader(code)
}

// Write is a no-op after claim() has been called.
func (tw *timeoutWriter) Write(b []byte) (int, error) {
	if tw.isClaimed() {
		return len(b), nil
	}
	return tw.ResponseWriter.Write(b)
}

// Header returns the underlying header map (always accessible).
func (tw *timeoutWriter) Header() http.Header {
	return tw.ResponseWriter.Header()
}
