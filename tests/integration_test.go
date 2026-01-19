package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gomarten/marten"
	"github.com/gomarten/marten/middleware"
)

// --- Integration Tests for v0.1.3 ---

// Test full CRUD API workflow
func TestCRUDWorkflow(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Logger, middleware.Recover)

	type User struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	users := make(map[string]User)
	nextID := 1

	// Create
	app.POST("/users", func(c *marten.Ctx) error {
		var user User
		if err := c.Bind(&user); err != nil {
			return c.BadRequest(err.Error())
		}
		user.ID = string(rune('0' + nextID))
		nextID++
		users[user.ID] = user
		return c.Created(user)
	})

	// Read
	app.GET("/users/:id", func(c *marten.Ctx) error {
		id := c.Param("id")
		user, exists := users[id]
		if !exists {
			return c.NotFound("user not found")
		}
		return c.OK(user)
	})

	// Update
	app.PUT("/users/:id", func(c *marten.Ctx) error {
		id := c.Param("id")
		user, exists := users[id]
		if !exists {
			return c.NotFound("user not found")
		}

		var update User
		if err := c.Bind(&update); err != nil {
			return c.BadRequest(err.Error())
		}

		user.Name = update.Name
		user.Email = update.Email
		users[id] = user
		return c.OK(user)
	})

	// Delete
	app.DELETE("/users/:id", func(c *marten.Ctx) error {
		id := c.Param("id")
		if _, exists := users[id]; !exists {
			return c.NotFound("user not found")
		}
		delete(users, id)
		return c.NoContent()
	})

	// Test workflow
	var userID string

	// 1. Create user
	createBody := `{"name":"Alice","email":"alice@example.com"}`
	req := httptest.NewRequest("POST", "/users", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 201 {
		t.Fatalf("create failed: %d - %s", rec.Code, rec.Body.String())
	}

	var created User
	json.Unmarshal(rec.Body.Bytes(), &created)
	userID = created.ID

	// 2. Read user
	req = httptest.NewRequest("GET", "/users/"+userID, nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("read failed: %d", rec.Code)
	}

	// 3. Update user
	updateBody := `{"name":"Alice Updated","email":"alice.new@example.com"}`
	req = httptest.NewRequest("PUT", "/users/"+userID, strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("update failed: %d", rec.Code)
	}

	// 4. Delete user
	req = httptest.NewRequest("DELETE", "/users/"+userID, nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 204 {
		t.Fatalf("delete failed: %d", rec.Code)
	}

	// 5. Verify deleted
	req = httptest.NewRequest("GET", "/users/"+userID, nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Fatalf("expected 404 after delete, got %d", rec.Code)
	}
}

// Test authentication flow
func TestAuthenticationFlow(t *testing.T) {
	app := marten.New()

	// Mock user database
	validToken := "valid-token-123"

	// Auth middleware
	authMiddleware := func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			token := c.Bearer()
			if token != validToken {
				return c.Unauthorized("invalid token")
			}
			c.Set("authenticated", true)
			return next(c)
		}
	}

	// Public route
	app.POST("/login", func(c *marten.Ctx) error {
		var creds struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.Bind(&creds); err != nil {
			return c.BadRequest(err.Error())
		}

		if creds.Username == "admin" && creds.Password == "secret" {
			return c.OK(marten.M{"token": validToken})
		}
		return c.Unauthorized("invalid credentials")
	})

	// Protected route
	app.GET("/profile", func(c *marten.Ctx) error {
		return c.OK(marten.M{"user": "admin", "email": "admin@example.com"})
	}, authMiddleware)

	// 1. Login
	loginBody := `{"username":"admin","password":"secret"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("login failed: %d", rec.Code)
	}

	// 2. Access protected route without token
	req = httptest.NewRequest("GET", "/profile", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 401 {
		t.Errorf("expected 401 without token, got %d", rec.Code)
	}

	// 3. Access protected route with token
	req = httptest.NewRequest("GET", "/profile", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200 with valid token, got %d", rec.Code)
	}
}

// Test file upload workflow
func TestFileUploadWorkflow(t *testing.T) {
	app := marten.New()
	app.Use(middleware.BodyLimit(5 * middleware.MB))

	uploadedFiles := make(map[string][]byte)

	app.POST("/upload", func(c *marten.Ctx) error {
		file, err := c.File("file")
		if err != nil {
			return c.BadRequest("no file uploaded")
		}

		f, err := file.Open()
		if err != nil {
			return c.ServerError("failed to open file")
		}
		defer f.Close()

		data := make([]byte, file.Size)
		_, err = f.Read(data)
		if err != nil {
			return c.ServerError("failed to read file")
		}

		uploadedFiles[file.Filename] = data
		return c.Created(marten.M{"filename": file.Filename, "size": file.Size})
	})

	app.GET("/files/:name", func(c *marten.Ctx) error {
		name := c.Param("name")
		data, exists := uploadedFiles[name]
		if !exists {
			return c.NotFound("file not found")
		}
		return c.Blob(200, "application/octet-stream", data)
	})

	// Create multipart form
	body := &bytes.Buffer{}
	body.WriteString("--boundary\r\n")
	body.WriteString("Content-Disposition: form-data; name=\"file\"; filename=\"test.txt\"\r\n")
	body.WriteString("Content-Type: text/plain\r\n\r\n")
	body.WriteString("Hello, World!\r\n")
	body.WriteString("--boundary--\r\n")

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 201 {
		t.Fatalf("upload failed: %d - %s", rec.Code, rec.Body.String())
	}

	// Download file
	req = httptest.NewRequest("GET", "/files/test.txt", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("download failed: %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "Hello, World!") {
		t.Errorf("file content mismatch: %s", rec.Body.String())
	}
}

// Test API versioning with groups
func TestAPIVersioning(t *testing.T) {
	app := marten.New()

	// V1 API
	v1 := app.Group("/api/v1")
	v1.GET("/users", func(c *marten.Ctx) error {
		return c.OK(marten.M{"version": "v1", "users": []string{"alice", "bob"}})
	})

	// V2 API
	v2 := app.Group("/api/v2")
	v2.GET("/users", func(c *marten.Ctx) error {
		return c.OK(marten.M{
			"version": "v2",
			"users": []marten.M{
				{"id": "1", "name": "alice"},
				{"id": "2", "name": "bob"},
			},
		})
	})

	// Test V1
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("v1 failed: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "\"version\":\"v1\"") {
		t.Error("v1 response incorrect")
	}

	// Test V2
	req = httptest.NewRequest("GET", "/api/v2/users", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("v2 failed: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "\"version\":\"v2\"") {
		t.Error("v2 response incorrect")
	}
}

// Test middleware stack with all built-in middleware
func TestFullMiddlewareStack(t *testing.T) {
	app := marten.New()

	// Add all middleware
	app.Use(middleware.RequestID)
	app.Use(middleware.Logger)
	app.Use(middleware.Recover)
	app.Use(middleware.CORS(middleware.DefaultCORSConfig()))
	app.Use(middleware.Secure(middleware.DefaultSecureConfig()))
	app.Use(middleware.NoCache)
	app.Use(middleware.BodyLimit(1 * middleware.MB))

	app.GET("/test", func(c *marten.Ctx) error {
		return c.OK(marten.M{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// Verify headers from middleware
	headers := []string{
		"X-Request-ID",
		"Access-Control-Allow-Origin",
		"X-Content-Type-Options",
		"Cache-Control",
	}

	for _, header := range headers {
		if rec.Header().Get(header) == "" {
			t.Errorf("missing header: %s", header)
		}
	}
}

// Test graceful error handling across the stack
func TestGracefulErrorHandling(t *testing.T) {
	app := marten.New()
	app.Use(middleware.Recover)

	var errorLog []string

	app.OnError(func(c *marten.Ctx, err error) {
		errorLog = append(errorLog, err.Error())
		c.JSON(500, marten.M{"error": err.Error()})
	})

	// Route that returns error
	app.GET("/error", func(c *marten.Ctx) error {
		return fmt.Errorf("custom error")
	})

	// Route that panics
	app.GET("/panic", func(c *marten.Ctx) error {
		panic("panic error")
	})

	// Test error
	req := httptest.NewRequest("GET", "/error", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 500 {
		t.Errorf("expected 500, got %d", rec.Code)
	}

	// Test panic
	req = httptest.NewRequest("GET", "/panic", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 500 {
		t.Errorf("expected 500 for panic, got %d", rec.Code)
	}
}

// Test complex routing scenarios
func TestComplexRouting(t *testing.T) {
	app := marten.New()

	// Mix of static, param, and wildcard routes
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "root")
	})
	app.GET("/about", func(c *marten.Ctx) error {
		return c.Text(200, "about")
	})
	app.GET("/users/:id", func(c *marten.Ctx) error {
		return c.Text(200, "user:"+c.Param("id"))
	})
	app.GET("/users/:id/posts/:postId", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("id")+":"+c.Param("postId"))
	})
	app.GET("/files/*path", func(c *marten.Ctx) error {
		return c.Text(200, "file:"+c.Param("path"))
	})
	app.GET("/api/v1/users", func(c *marten.Ctx) error {
		return c.Text(200, "api-users")
	})

	tests := []struct {
		path     string
		expected string
	}{
		{"/", "root"},
		{"/about", "about"},
		{"/users/123", "user:123"},
		{"/users/456/posts/789", "456:789"},
		{"/files/doc.pdf", "file:doc.pdf"},
		{"/files/images/photo.png", "file:images/photo.png"},
		{"/api/v1/users", "api-users"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != tt.expected {
			t.Errorf("path %s: expected %q, got %q", tt.path, tt.expected, rec.Body.String())
		}
	}
}
