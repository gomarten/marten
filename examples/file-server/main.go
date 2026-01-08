// File Server example - Static file serving with wildcard routes
package main

import (
	"io"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/gomarten/marten"
	"github.com/gomarten/marten/middleware"
)

func main() {
	app := marten.New()

	app.Use(middleware.Logger, middleware.Recover)

	// Serve static files using wildcard route
	app.GET("/static/*filepath", serveStatic("./public"))

	// Serve uploads
	app.GET("/uploads/*filepath", serveStatic("./uploads"))

	// API routes take precedence over wildcards
	app.GET("/api/files", func(c *marten.Ctx) error {
		files, _ := listFiles("./public")
		return c.OK(marten.M{"files": files})
	})

	// Download with custom filename
	app.GET("/download/:filename", func(c *marten.Ctx) error {
		filename := c.Param("filename")
		path := filepath.Join("./public", filename)

		// Security: prevent directory traversal
		if strings.Contains(filename, "..") {
			return c.BadRequest("invalid filename")
		}

		file, err := os.Open(path)
		if err != nil {
			return c.NotFound("file not found")
		}
		defer file.Close()

		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Header("Content-Type", "application/octet-stream")
		c.Status(200)
		io.Copy(c.Writer, file)
		return nil
	})

	// SPA fallback - serve index.html for unmatched routes
	app.NotFound(func(c *marten.Ctx) error {
		// Check if it's an API request
		if strings.HasPrefix(c.Path(), "/api/") {
			return c.NotFound("endpoint not found")
		}

		// Serve index.html for SPA
		return serveFile(c, "./public/index.html")
	})

	log.Println("File server running on http://localhost:3000")
	log.Println("Serving files from ./public at /static/*")
	app.Run(":3000")
}

// serveStatic returns a handler that serves static files
func serveStatic(root string) marten.Handler {
	return func(c *marten.Ctx) error {
		filepath := c.Param("filepath")

		// Security: prevent directory traversal
		if strings.Contains(filepath, "..") {
			return c.BadRequest("invalid path")
		}

		path := root + "/" + filepath
		return serveFile(c, path)
	}
}

// serveFile serves a single file
func serveFile(c *marten.Ctx, path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return c.NotFound("file not found")
		}
		return c.ServerError("failed to open file")
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return c.ServerError("failed to stat file")
	}

	if stat.IsDir() {
		// Try index.html
		indexPath := filepath.Join(path, "index.html")
		return serveFile(c, indexPath)
	}

	// Set content type based on extension
	ext := filepath.Ext(path)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)

	// Serve file
	c.Status(200)
	io.Copy(c.Writer, file)
	return nil
}

// listFiles returns a list of files in a directory
func listFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(dir, path)
			files = append(files, rel)
		}
		return nil
	})
	return files, err
}
