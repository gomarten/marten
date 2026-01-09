package middleware

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"net/http"

	"github.com/gomarten/marten"
)

// ETag returns a middleware that adds ETag headers for caching.
func ETag(next marten.Handler) marten.Handler {
	return func(c *marten.Ctx) error {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			return next(c)
		}

		origWriter := c.Writer
		ew := &etagWriter{
			ResponseWriter: origWriter,
			buf:            &bytes.Buffer{},
		}
		c.Writer = ew

		err := next(c)
		c.Writer = origWriter

		if ew.status >= 200 && ew.status < 300 && ew.buf.Len() > 0 {
			hash := sha1.Sum(ew.buf.Bytes())
			etag := `"` + hex.EncodeToString(hash[:8]) + `"`

			if match := c.Request.Header.Get("If-None-Match"); match == etag {
				origWriter.WriteHeader(http.StatusNotModified)
				return nil
			}

			origWriter.Header().Set("ETag", etag)
			origWriter.WriteHeader(ew.status)
			_, _ = origWriter.Write(ew.buf.Bytes())
			return err
		}

		if ew.status == 0 {
			ew.status = http.StatusOK
		}
		origWriter.WriteHeader(ew.status)
		_, _ = origWriter.Write(ew.buf.Bytes())
		return err
	}
}

type etagWriter struct {
	http.ResponseWriter
	buf    *bytes.Buffer
	status int
}

func (w *etagWriter) WriteHeader(code int) {
	w.status = code
}

func (w *etagWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.buf.Write(b)
}
