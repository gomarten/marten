// File Server example - Static file serving with built-in middleware
package main

import (
	"log"
	"strings"

	"github.com/gomarten/marten"
	"github.com/gomarten/marten/middleware"
)

func main() {
	app := marten.New()

	app.Use(middleware.Logger, middleware.Recover)

	// API routes first (before static middleware)
	app.GET("/api/files", func(c *marten.Ctx) error {
		return c.OK(marten.M{
			"message": "File API endpoint",
			"files":   []string{"example1.txt", "example2.txt"},
		})
	})

	// Serve static files using built-in middleware
	// This is the recommended approach for serving static files
	app.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:   "./public",
		Prefix: "/static",
		MaxAge: 3600, // Cache for 1 hour
		Browse: false, // Disable directory browsing for security
	}))

	// Serve uploads from a different directory
	app.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:   "./uploads",
		Prefix: "/uploads",
		MaxAge: 86400, // Cache for 24 hours
	}))

	// SPA fallback - serve index.html for unmatched routes
	app.NotFound(func(c *marten.Ctx) error {
		// API routes return 404
		if strings.HasPrefix(c.Path(), "/api/") {
			return c.NotFound("endpoint not found")
		}

		// Serve index.html for SPA client-side routing
		// Note: In production, you might want to use a proper file serving approach
		return c.HTML(200, `<!DOCTYPE html>
<html>
<head>
	<title>File Server Example</title>
</head>
<body>
	<h1>File Server Example</h1>
	<p>This is a placeholder index.html for SPA fallback.</p>
	<p>In production, replace this with your actual SPA index.html file.</p>
	<ul>
		<li><a href="/static/">Browse static files</a></li>
		<li><a href="/uploads/">Browse uploads</a></li>
		<li><a href="/api/files">API endpoint</a></li>
	</ul>
</body>
</html>`)
	})

	log.Println("File server running on http://localhost:3000")
	log.Println("Static files: http://localhost:3000/static/")
	log.Println("Uploads: http://localhost:3000/uploads/")
	log.Println("API: http://localhost:3000/api/files")
	app.Run(":3000")
}
