package middleware

import (
	"errors"
	"io"
	"net/http"

	"github.com/gomarten/marten"
)

// Size constants
const (
	KB int64 = 1024
	MB int64 = 1024 * KB
	GB int64 = 1024 * MB
)

// BodyLimit returns a middleware that limits request body size.
// It checks Content-Length first (when available), then enforces the limit
// during the actual read so chunked-encoded bodies are covered too.
// When the limit is exceeded during a read, the handler receives a
// *bodyTooLargeError from io.ReadAll / json.Decode; BodyLimit catches that
// error on the way back out and responds with 413 before it reaches OnError.
func BodyLimit(maxSize int64) marten.Middleware {
	return func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			// Fast path: Content-Length is known and already over limit.
			if c.Request.ContentLength > maxSize {
				return c.JSON(http.StatusRequestEntityTooLarge, marten.E("request body too large"))
			}

			// Wrap body to enforce limit during streaming reads.
			c.Request.Body = &limitedReader{
				reader:  c.Request.Body,
				maxSize: maxSize,
			}

			err := next(c)

			// If the handler returned a bodyTooLargeError (from a read inside
			// the handler) and the response hasn't been written yet, send 413.
			var tooLarge *bodyTooLargeError
			if errors.As(err, &tooLarge) && !c.Written() {
				return c.JSON(http.StatusRequestEntityTooLarge, marten.E("request body too large"))
			}

			return err
		}
	}
}

type limitedReader struct {
	reader  io.ReadCloser
	maxSize int64
	read    int64
}

func (r *limitedReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.read += int64(n)
	if r.read > r.maxSize {
		return n, &bodyTooLargeError{}
	}
	return n, err
}

func (r *limitedReader) Close() error {
	return r.reader.Close()
}

type bodyTooLargeError struct{}

func (e *bodyTooLargeError) Error() string {
	return "request body too large"
}
