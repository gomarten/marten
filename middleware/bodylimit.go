package middleware

import (
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
func BodyLimit(maxSize int64) marten.Middleware {
	return func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			// Check Content-Length if available (skip for chunked encoding where ContentLength = -1)
			if c.Request.ContentLength > maxSize {
				return c.JSON(http.StatusRequestEntityTooLarge, marten.E("request body too large"))
			}

			// Always wrap body to enforce limit during read (handles chunked encoding)
			c.Request.Body = &limitedReader{
				reader:  c.Request.Body,
				maxSize: maxSize,
			}

			return next(c)
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
