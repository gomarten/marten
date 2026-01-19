package tests

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gomarten/marten"
	"github.com/gomarten/marten/middleware"
)

// Setup test directory structure
func setupTestFiles(t *testing.T) string {
	tmpDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"index.html":         "<html><body>Index</body></html>",
		"about.html":         "<html><body>About</body></html>",
		"style.css":          "body { margin: 0; }",
		"script.js":          "console.log('test');",
		"image.png":          "fake-png-data",
		"subdir/nested.html": "<html><body>Nested</body></html>",
		"subdir/index.html":  "<html><body>Subdir Index</body></html>",
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)
		
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	return tmpDir
}

// Test basic static file serving
func TestStaticBasicServing(t *testing.T) {
	tmpDir := setupTestFiles(t)
	
	app := marten.New()
	app.Use(middleware.Static(tmpDir))

	tests := []struct {
		path        string
		expectedCode int
		contains    string
	}{
		{"/index.html", 200, "Index"},
		{"/about.html", 200, "About"},
		{"/style.css", 200, "margin"},
		{"/script.js", 200, "console.log"},
		{"/nonexistent.html", 404, ""},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code != tt.expectedCode {
			t.Errorf("path %s: expected %d, got %d", tt.path, tt.expectedCode, rec.Code)
		}

		if tt.contains != "" && !strings.Contains(rec.Body.String(), tt.contains) {
			t.Errorf("path %s: expected body to contain %q, got %q", tt.path, tt.contains, rec.Body.String())
		}
	}
}

// Test directory index serving
func TestStaticDirectoryIndex(t *testing.T) {
	tmpDir := setupTestFiles(t)
	
	app := marten.New()
	app.Use(middleware.Static(tmpDir))

	// Root directory should serve index.html
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Index") {
		t.Errorf("expected index.html content, got %q", rec.Body.String())
	}

	// Subdirectory should serve its index.html
	req = httptest.NewRequest("GET", "/subdir/", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Subdir Index") {
		t.Errorf("expected subdir index.html content, got %q", rec.Body.String())
	}
}

// Test nested file serving
func TestStaticNestedFiles(t *testing.T) {
	tmpDir := setupTestFiles(t)
	
	app := marten.New()
	app.Use(middleware.Static(tmpDir))

	req := httptest.NewRequest("GET", "/subdir/nested.html", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Nested") {
		t.Errorf("expected nested content, got %q", rec.Body.String())
	}
}

// Test directory traversal prevention
func TestStaticDirectoryTraversalPrevention(t *testing.T) {
	tmpDir := setupTestFiles(t)
	
	app := marten.New()
	app.Use(middleware.Static(tmpDir))

	tests := []string{
		"/../etc/passwd",
		"/subdir/../../etc/passwd",
		"/./../../etc/passwd",
		"/../../../etc/passwd",
	}

	for _, path := range tests {
		req := httptest.NewRequest("GET", path, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Code == 200 {
			t.Errorf("path %s: directory traversal should be prevented", path)
		}
	}
}

// Test content type headers
func TestStaticContentTypes(t *testing.T) {
	tmpDir := setupTestFiles(t)
	
	app := marten.New()
	app.Use(middleware.Static(tmpDir))

	tests := []struct {
		path        string
		contentType string
	}{
		{"/index.html", "text/html"},
		{"/style.css", "text/css"},
		{"/script.js", "text/javascript"},
		{"/image.png", "image/png"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		ct := rec.Header().Get("Content-Type")
		if !strings.HasPrefix(ct, tt.contentType) {
			t.Errorf("path %s: expected content-type %s, got %s", tt.path, tt.contentType, ct)
		}
	}
}

// Test with prefix configuration
func TestStaticWithPrefix(t *testing.T) {
	tmpDir := setupTestFiles(t)
	
	app := marten.New()
	app.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:   tmpDir,
		Prefix: "/static",
	}))

	// Should serve with prefix
	req := httptest.NewRequest("GET", "/static/index.html", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// Should not serve without prefix
	req = httptest.NewRequest("GET", "/index.html", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Errorf("expected 404 without prefix, got %d", rec.Code)
	}
}

// Test directory browsing
func TestStaticDirectoryBrowsing(t *testing.T) {
	tmpDir := setupTestFiles(t)
	
	app := marten.New()
	app.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:   tmpDir,
		Browse: true,
	}))

	// Directory without index should show listing
	emptyDir := filepath.Join(tmpDir, "empty")
	os.Mkdir(emptyDir, 0755)

	req := httptest.NewRequest("GET", "/empty/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Index of") {
		t.Errorf("expected directory listing, got %q", rec.Body.String())
	}
}

// Test If-Modified-Since header
func TestStaticIfModifiedSince(t *testing.T) {
	tmpDir := setupTestFiles(t)
	
	app := marten.New()
	app.Use(middleware.Static(tmpDir))

	// First request to get Last-Modified
	req := httptest.NewRequest("GET", "/index.html", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	lastModified := rec.Header().Get("Last-Modified")
	if lastModified == "" {
		t.Fatal("expected Last-Modified header")
	}

	// Second request with If-Modified-Since
	req = httptest.NewRequest("GET", "/index.html", nil)
	req.Header.Set("If-Modified-Since", lastModified)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 304 {
		t.Errorf("expected 304 Not Modified, got %d", rec.Code)
	}
}

// Test HEAD requests
func TestStaticHEADRequests(t *testing.T) {
	tmpDir := setupTestFiles(t)
	
	app := marten.New()
	app.Use(middleware.Static(tmpDir))

	req := httptest.NewRequest("HEAD", "/index.html", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("HEAD request should have empty body, got %d bytes", rec.Body.Len())
	}
	if rec.Header().Get("Content-Type") == "" {
		t.Error("expected Content-Type header")
	}
}

// Test POST requests are ignored
func TestStaticIgnoresPOST(t *testing.T) {
	tmpDir := setupTestFiles(t)
	
	app := marten.New()
	app.Use(middleware.Static(tmpDir))
	app.POST("/index.html", func(c *marten.Ctx) error {
		return c.Text(200, "POST handler")
	})

	req := httptest.NewRequest("POST", "/index.html", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "POST handler" {
		t.Errorf("expected POST handler to be called, got %q", rec.Body.String())
	}
}

// Test custom NotFoundHandler
func TestStaticCustomNotFoundHandler(t *testing.T) {
	tmpDir := setupTestFiles(t)
	
	app := marten.New()
	app.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root: tmpDir,
		NotFoundHandler: func(c *marten.Ctx) error {
			return c.Text(404, "Custom 404")
		},
	}))

	req := httptest.NewRequest("GET", "/nonexistent.html", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "Custom 404" {
		t.Errorf("expected custom 404 handler, got %q", rec.Body.String())
	}
}

// Test static middleware with API routes
func TestStaticWithAPIRoutes(t *testing.T) {
	tmpDir := setupTestFiles(t)
	
	app := marten.New()
	
	// API routes should take precedence
	app.GET("/api/users", func(c *marten.Ctx) error {
		return c.OK(marten.M{"users": []string{"alice", "bob"}})
	})
	
	// Static files
	app.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Root:   tmpDir,
		Prefix: "/static",
	}))

	// Test API route
	req := httptest.NewRequest("GET", "/api/users", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("API route: expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "alice") {
		t.Error("API route should return JSON")
	}

	// Test static file
	req = httptest.NewRequest("GET", "/static/index.html", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("Static file: expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Index") {
		t.Error("Static file should return HTML")
	}
}
