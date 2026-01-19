package middleware

import (
	"fmt"
	"html"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gomarten/marten"
)

// StaticConfig configures the static file serving middleware.
type StaticConfig struct {
	// Root is the root directory to serve files from (required)
	Root string
	// Index is the index file to serve for directories (default: "index.html")
	Index string
	// Browse enables directory browsing (default: false)
	Browse bool
	// MaxAge sets Cache-Control max-age in seconds (default: 0 = no cache)
	MaxAge int
	// Prefix is the URL prefix to strip before looking up files (optional)
	Prefix string
	// NotFoundHandler is called when file is not found (optional)
	NotFoundHandler marten.Handler
	// SkipLogging skips logging for static files (default: false)
	SkipLogging bool
}

// DefaultStaticConfig returns sensible defaults.
func DefaultStaticConfig(root string) StaticConfig {
	return StaticConfig{
		Root:        root,
		Index:       "index.html",
		Browse:      false,
		MaxAge:      0,
		SkipLogging: false,
	}
}

// Static returns a middleware that serves static files from the specified directory.
// This is a convenience function that uses DefaultStaticConfig.
func Static(root string) marten.Middleware {
	return StaticWithConfig(DefaultStaticConfig(root))
}

// StaticWithConfig returns a static file serving middleware with custom config.
func StaticWithConfig(cfg StaticConfig) marten.Middleware {
	// Validate config
	if cfg.Root == "" {
		panic("static middleware: root directory is required")
	}
	if cfg.Index == "" {
		cfg.Index = "index.html"
	}

	// Clean root path
	cfg.Root = filepath.Clean(cfg.Root)

	return func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			// Only handle GET and HEAD requests
			if c.Method() != http.MethodGet && c.Method() != http.MethodHead {
				return next(c)
			}

			// Get the file path
			path := c.Path()

			// Strip prefix if configured
			if cfg.Prefix != "" {
				if !strings.HasPrefix(path, cfg.Prefix) {
					return next(c)
				}
				path = strings.TrimPrefix(path, cfg.Prefix)
			}

			// Security: prevent directory traversal
			if strings.Contains(path, "..") {
				return next(c)
			}

			// Build full file path
			filePath := filepath.Join(cfg.Root, filepath.Clean(path))

			// Check if file exists
			stat, err := os.Stat(filePath)
			if err != nil {
				if os.IsNotExist(err) {
					if cfg.NotFoundHandler != nil {
						return cfg.NotFoundHandler(c)
					}
					return next(c)
				}
				return next(c)
			}

			// Handle directories
			if stat.IsDir() {
				// Try index file
				indexPath := filepath.Join(filePath, cfg.Index)
				if indexStat, err := os.Stat(indexPath); err == nil && !indexStat.IsDir() {
					return serveFile(c, indexPath, cfg)
				}

				// Directory browsing
				if cfg.Browse {
					return serveDirListing(c, filePath, path)
				}

				// No index and browsing disabled
				if cfg.NotFoundHandler != nil {
					return cfg.NotFoundHandler(c)
				}
				return next(c)
			}

			// Serve the file
			return serveFile(c, filePath, cfg)
		}
	}
}

// serveFile serves a single file with proper headers.
func serveFile(c *marten.Ctx, filePath string, cfg StaticConfig) error {
	file, err := os.Open(filePath)
	if err != nil {
		return c.ServerError("failed to open file")
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return c.ServerError("failed to stat file")
	}

	// Set content type based on extension
	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)

	// Set Content-Length for download progress
	c.Header("Content-Length", fmt.Sprintf("%d", stat.Size()))

	// Set cache headers
	if cfg.MaxAge > 0 {
		c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", cfg.MaxAge))
	} else {
		c.Header("Cache-Control", "no-cache")
	}

	// Set Last-Modified header
	c.Header("Last-Modified", stat.ModTime().UTC().Format(http.TimeFormat))

	// Check If-Modified-Since header
	if modifiedSince := c.GetHeader("If-Modified-Since"); modifiedSince != "" {
		if t, err := http.ParseTime(modifiedSince); err == nil {
			// Truncate both times to seconds for comparison
			modTime := stat.ModTime().Truncate(1000000000) // 1 second in nanoseconds
			reqTime := t.Truncate(1000000000)
			if modTime.Equal(reqTime) || modTime.Before(reqTime) {
				c.Status(http.StatusNotModified)
				return nil
			}
		}
	}

	// For HEAD requests, don't send body
	if c.Method() == http.MethodHead {
		c.Status(http.StatusOK)
		return nil
	}

	// Serve file
	return c.Stream(http.StatusOK, contentType, file)
}

// serveDirListing serves a directory listing.
func serveDirListing(c *marten.Ctx, dirPath, urlPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return c.ServerError("failed to read directory")
	}

	// Escape HTML to prevent XSS
	escapedPath := html.EscapeString(urlPath)

	// Build HTML response
	htmlContent := `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<title>Index of ` + escapedPath + `</title>
	<style>
		body { font-family: monospace; margin: 2em; }
		h1 { border-bottom: 1px solid #ccc; padding-bottom: 0.5em; }
		ul { list-style: none; padding: 0; }
		li { padding: 0.5em 0; }
		a { text-decoration: none; color: #0066cc; }
		a:hover { text-decoration: underline; }
		.dir { font-weight: bold; }
		.size { color: #666; margin-left: 1em; }
	</style>
</head>
<body>
	<h1>Index of ` + escapedPath + `</h1>
	<ul>`

	// Add parent directory link if not root
	if urlPath != "/" {
		parentPath := path.Dir(urlPath)
		if parentPath == "." {
			parentPath = "/"
		}
		htmlContent += `<li><a href="` + parentPath + `" class="dir">../</a></li>`
	}

	// Add entries
	for _, entry := range entries {
		name := entry.Name()
		escapedName := html.EscapeString(name)
		info, _ := entry.Info()
		
		if entry.IsDir() {
			// Use path.Join for URL construction (not filepath.Join which uses OS separators)
			dirURL := path.Join(urlPath, name) + "/"
			htmlContent += `<li><a href="` + dirURL + `" class="dir">` + escapedName + `/</a></li>`
		} else {
			size := ""
			if info != nil {
				size = formatSize(info.Size())
			}
			// Use path.Join for URL construction
			fileURL := path.Join(urlPath, name)
			htmlContent += `<li><a href="` + fileURL + `">` + escapedName + `</a><span class="size">` + size + `</span></li>`
		}
	}

	htmlContent += `</ul>
</body>
</html>`

	return c.HTML(http.StatusOK, htmlContent)
}

// formatSize formats file size in human-readable format.
func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	units := []string{"KB", "MB", "GB", "TB"}
	if exp >= len(units) {
		exp = len(units) - 1
	}
	return fmt.Sprintf("%.1f %s", float64(size)/float64(div), units[exp])
}
