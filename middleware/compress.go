package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gomarten/marten"
)

// CompressConfig configures compression middleware.
type CompressConfig struct {
	Level        int
	MinSize      int
	ContentTypes []string
}

// DefaultCompressConfig returns sensible defaults.
func DefaultCompressConfig() CompressConfig {
	return CompressConfig{
		Level:   gzip.DefaultCompression,
		MinSize: 1024,
		ContentTypes: []string{
			"text/plain",
			"text/html",
			"text/css",
			"text/javascript",
			"application/json",
			"application/javascript",
			"application/xml",
		},
	}
}

var gzipPool = sync.Pool{
	New: func() any {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		return w
	},
}

// Compress returns a gzip compression middleware.
func Compress(cfg CompressConfig) marten.Middleware {
	if cfg.MinSize == 0 {
		cfg.MinSize = 1024
	}
	if cfg.Level == 0 {
		cfg.Level = gzip.DefaultCompression
	}

	return func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			if !strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip") {
				return next(c)
			}

			origWriter := c.Writer
			gw := &gzipResponseWriter{
				ResponseWriter: origWriter,
				cfg:            cfg,
			}
			c.Writer = gw

			err := next(c)

			// Restore original writer
			c.Writer = origWriter

			// Flush any remaining buffered data
			if gw.gw != nil {
				gw.gw.Close()
				gzipPool.Put(gw.gw)
			} else if len(gw.buf) > 0 {
				// Write buffered data that didn't reach MinSize
				_, _ = origWriter.Write(gw.buf)
			}

			return err
		}
	}
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gw          *gzip.Writer
	cfg         CompressConfig
	wroteHeader bool
	buf         []byte
}

func (w *gzipResponseWriter) shouldCompress() bool {
	ct := w.Header().Get("Content-Type")
	if ct == "" {
		return false
	}
	for _, allowed := range w.cfg.ContentTypes {
		if strings.HasPrefix(ct, allowed) {
			return true
		}
	}
	return false
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	if !w.shouldCompress() {
		w.ResponseWriter.WriteHeader(code)
		return
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	if w.gw == nil && len(w.buf)+len(b) < w.cfg.MinSize {
		w.buf = append(w.buf, b...)
		return len(b), nil
	}

	if w.gw == nil && w.shouldCompress() {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")
		w.gw = gzipPool.Get().(*gzip.Writer)
		w.gw.Reset(w.ResponseWriter)

		if len(w.buf) > 0 {
			_, _ = w.gw.Write(w.buf)
			w.buf = nil
		}
	}

	if w.gw != nil {
		return w.gw.Write(b)
	}

	if len(w.buf) > 0 {
		_, _ = w.ResponseWriter.Write(w.buf)
		w.buf = nil
	}
	return w.ResponseWriter.Write(b)
}

func (w *gzipResponseWriter) Flush() {
	// Flush any buffered data without compression if under MinSize
	if w.gw == nil && len(w.buf) > 0 {
		_, _ = w.ResponseWriter.Write(w.buf)
		w.buf = nil
	}
	if w.gw != nil {
		w.gw.Flush()
	}
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Close flushes any remaining buffered data.
func (w *gzipResponseWriter) Close() error {
	if w.gw == nil && len(w.buf) > 0 {
		_, _ = w.ResponseWriter.Write(w.buf)
		w.buf = nil
	}
	return nil
}
