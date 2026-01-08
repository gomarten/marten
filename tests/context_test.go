package tests

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gomarten/marten"
)

func TestContextText(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "hello world")
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Errorf("expected text/plain content type, got %q", ct)
	}
	if rec.Body.String() != "hello world" {
		t.Errorf("expected 'hello world', got %q", rec.Body.String())
	}
}

func TestContextJSON(t *testing.T) {
	type response struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		return c.JSON(200, response{Name: "Alice", Age: 30})
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("expected application/json content type, got %q", ct)
	}

	var resp response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Name != "Alice" || resp.Age != 30 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestContextBind(t *testing.T) {
	type input struct {
		Name string `json:"name"`
	}

	app := marten.New()
	app.POST("/", func(c *marten.Ctx) error {
		var in input
		if err := c.Bind(&in); err != nil {
			return c.Text(400, "bad request")
		}
		return c.Text(200, "hello "+in.Name)
	})

	body := bytes.NewBufferString(`{"name":"Bob"}`)
	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "hello Bob" {
		t.Errorf("expected 'hello Bob', got %q", rec.Body.String())
	}
}

func TestContextQuery(t *testing.T) {
	app := marten.New()
	app.GET("/search", func(c *marten.Ctx) error {
		q := c.Query("q")
		page := c.Query("page")
		return c.Text(200, q+":"+page)
	})

	req := httptest.NewRequest("GET", "/search?q=golang&page=2", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "golang:2" {
		t.Errorf("expected 'golang:2', got %q", rec.Body.String())
	}
}

func TestContextHeader(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		c.Header("X-Custom", "value")
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if h := rec.Header().Get("X-Custom"); h != "value" {
		t.Errorf("expected X-Custom header 'value', got %q", h)
	}
}

func TestContextParam(t *testing.T) {
	app := marten.New()
	app.GET("/users/:id/posts/:postId", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("id")+"-"+c.Param("postId"))
	})

	req := httptest.NewRequest("GET", "/users/42/posts/7", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "42-7" {
		t.Errorf("expected '42-7', got %q", rec.Body.String())
	}
}

func TestContextStatus(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		c.Status(http.StatusAccepted)
		return nil
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 202 {
		t.Errorf("expected 202, got %d", rec.Code)
	}
}

// --- New Feature Tests ---

func TestContextParamInt(t *testing.T) {
	app := marten.New()
	app.GET("/users/:id", func(c *marten.Ctx) error {
		id := c.ParamInt("id")
		return c.JSON(200, map[string]int{"id": id})
	})

	tests := []struct {
		path     string
		expected int
	}{
		{"/users/42", 42},
		{"/users/0", 0},
		{"/users/abc", 0}, // invalid returns 0
		{"/users/-5", -5},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		var resp map[string]int
		json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["id"] != tt.expected {
			t.Errorf("%s: expected id=%d, got %d", tt.path, tt.expected, resp["id"])
		}
	}
}

func TestContextParamInt64(t *testing.T) {
	app := marten.New()
	app.GET("/items/:id", func(c *marten.Ctx) error {
		id := c.ParamInt64("id")
		return c.JSON(200, map[string]int64{"id": id})
	})

	req := httptest.NewRequest("GET", "/items/9223372036854775807", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var resp map[string]int64
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["id"] != 9223372036854775807 {
		t.Errorf("expected max int64, got %d", resp["id"])
	}
}

func TestContextQueryInt(t *testing.T) {
	app := marten.New()
	app.GET("/search", func(c *marten.Ctx) error {
		page := c.QueryInt("page")
		limit := c.QueryInt("limit")
		return c.JSON(200, map[string]int{"page": page, "limit": limit})
	})

	req := httptest.NewRequest("GET", "/search?page=5&limit=20", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var resp map[string]int
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["page"] != 5 || resp["limit"] != 20 {
		t.Errorf("expected page=5, limit=20, got %v", resp)
	}
}

func TestContextQueryDefault(t *testing.T) {
	app := marten.New()
	app.GET("/search", func(c *marten.Ctx) error {
		sort := c.QueryDefault("sort", "created_at")
		order := c.QueryDefault("order", "desc")
		return c.Text(200, sort+":"+order)
	})

	// With defaults
	req := httptest.NewRequest("GET", "/search", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Body.String() != "created_at:desc" {
		t.Errorf("expected defaults, got %q", rec.Body.String())
	}

	// With overrides
	req = httptest.NewRequest("GET", "/search?sort=name&order=asc", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Body.String() != "name:asc" {
		t.Errorf("expected overrides, got %q", rec.Body.String())
	}
}

func TestContextOK(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		return c.OK(map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}
}

func TestContextCreated(t *testing.T) {
	app := marten.New()
	app.POST("/users", func(c *marten.Ctx) error {
		return c.Created(map[string]string{"id": "123"})
	})

	req := httptest.NewRequest("POST", "/users", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 201 {
		t.Errorf("expected 201, got %d", rec.Code)
	}
}

func TestContextNoContent(t *testing.T) {
	app := marten.New()
	app.DELETE("/users/:id", func(c *marten.Ctx) error {
		return c.NoContent()
	})

	req := httptest.NewRequest("DELETE", "/users/123", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 204 {
		t.Errorf("expected 204, got %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("expected empty body, got %q", rec.Body.String())
	}
}

func TestContextErrorResponses(t *testing.T) {
	tests := []struct {
		name     string
		handler  func(*marten.Ctx) error
		expected int
		message  string
	}{
		{"BadRequest", func(c *marten.Ctx) error { return c.BadRequest("bad") }, 400, "bad"},
		{"Unauthorized", func(c *marten.Ctx) error { return c.Unauthorized("unauth") }, 401, "unauth"},
		{"Forbidden", func(c *marten.Ctx) error { return c.Forbidden("forbidden") }, 403, "forbidden"},
		{"NotFound", func(c *marten.Ctx) error { return c.NotFound("not found") }, 404, "not found"},
		{"ServerError", func(c *marten.Ctx) error { return c.ServerError("error") }, 500, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := marten.New()
			app.GET("/", tt.handler)

			req := httptest.NewRequest("GET", "/", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Code != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, rec.Code)
			}

			var resp map[string]string
			json.Unmarshal(rec.Body.Bytes(), &resp)
			if resp["error"] != tt.message {
				t.Errorf("expected error=%q, got %q", tt.message, resp["error"])
			}
		})
	}
}

func TestContextRedirect(t *testing.T) {
	app := marten.New()
	app.GET("/old", func(c *marten.Ctx) error {
		return c.Redirect(301, "/new")
	})

	req := httptest.NewRequest("GET", "/old", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 301 {
		t.Errorf("expected 301, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/new" {
		t.Errorf("expected Location=/new, got %q", loc)
	}
}

func TestContextClientIP(t *testing.T) {
	app := marten.New()
	app.GET("/ip", func(c *marten.Ctx) error {
		return c.Text(200, c.ClientIP())
	})

	tests := []struct {
		name     string
		headers  map[string]string
		expected string
	}{
		{
			name:     "X-Forwarded-For single",
			headers:  map[string]string{"X-Forwarded-For": "203.0.113.195"},
			expected: "203.0.113.195",
		},
		{
			name:     "X-Forwarded-For multiple",
			headers:  map[string]string{"X-Forwarded-For": "203.0.113.195, 70.41.3.18, 150.172.238.178"},
			expected: "203.0.113.195",
		},
		{
			name:     "X-Real-IP",
			headers:  map[string]string{"X-Real-IP": "203.0.113.195"},
			expected: "203.0.113.195",
		},
		{
			name:     "X-Forwarded-For takes precedence",
			headers:  map[string]string{"X-Forwarded-For": "1.1.1.1", "X-Real-IP": "2.2.2.2"},
			expected: "1.1.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ip", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Body.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, rec.Body.String())
			}
		})
	}
}

func TestContextBearer(t *testing.T) {
	app := marten.New()
	app.GET("/token", func(c *marten.Ctx) error {
		return c.Text(200, c.Bearer())
	})

	tests := []struct {
		auth     string
		expected string
	}{
		{"Bearer abc123", "abc123"},
		{"Bearer ", ""},
		{"Basic abc123", ""},
		{"", ""},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/token", nil)
		if tt.auth != "" {
			req.Header.Set("Authorization", tt.auth)
		}
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != tt.expected {
			t.Errorf("auth=%q: expected %q, got %q", tt.auth, tt.expected, rec.Body.String())
		}
	}
}

func TestContextIsJSON(t *testing.T) {
	app := marten.New()
	app.POST("/check", func(c *marten.Ctx) error {
		if c.IsJSON() {
			return c.Text(200, "json")
		}
		return c.Text(200, "not json")
	})

	tests := []struct {
		contentType string
		expected    string
	}{
		{"application/json", "json"},
		{"application/json; charset=utf-8", "json"},
		{"text/plain", "not json"},
		{"", "not json"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("POST", "/check", nil)
		if tt.contentType != "" {
			req.Header.Set("Content-Type", tt.contentType)
		}
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if rec.Body.String() != tt.expected {
			t.Errorf("Content-Type=%q: expected %q, got %q", tt.contentType, tt.expected, rec.Body.String())
		}
	}
}

func TestContextIsAJAX(t *testing.T) {
	app := marten.New()
	app.GET("/check", func(c *marten.Ctx) error {
		if c.IsAJAX() {
			return c.Text(200, "ajax")
		}
		return c.Text(200, "not ajax")
	})

	// AJAX request
	req := httptest.NewRequest("GET", "/check", nil)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Body.String() != "ajax" {
		t.Errorf("expected ajax, got %q", rec.Body.String())
	}

	// Non-AJAX request
	req = httptest.NewRequest("GET", "/check", nil)
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Body.String() != "not ajax" {
		t.Errorf("expected not ajax, got %q", rec.Body.String())
	}
}

func TestContextRequestID(t *testing.T) {
	app := marten.New()
	app.GET("/id", func(c *marten.Ctx) error {
		id1 := c.RequestID()
		id2 := c.RequestID() // Should return same ID
		if id1 != id2 {
			return c.Text(500, "IDs don't match")
		}
		return c.Text(200, id1)
	})

	// Auto-generated ID
	req := httptest.NewRequest("GET", "/id", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if len(rec.Body.String()) != 16 { // 8 bytes = 16 hex chars
		t.Errorf("expected 16 char ID, got %q", rec.Body.String())
	}

	// Provided ID
	req = httptest.NewRequest("GET", "/id", nil)
	req.Header.Set("X-Request-ID", "custom-id-123")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Body.String() != "custom-id-123" {
		t.Errorf("expected custom-id-123, got %q", rec.Body.String())
	}
}

func TestContextStore(t *testing.T) {
	app := marten.New()

	// Middleware that sets values
	app.Use(func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			c.Set("user", "alice")
			c.Set("role", "admin")
			return next(c)
		}
	})

	app.GET("/", func(c *marten.Ctx) error {
		user := c.GetString("user")
		role := c.GetString("role")
		missing := c.GetString("missing") // Should return ""
		return c.Text(200, user+":"+role+":"+missing)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "alice:admin:" {
		t.Errorf("expected alice:admin:, got %q", rec.Body.String())
	}
}

func TestContextGet(t *testing.T) {
	app := marten.New()

	app.Use(func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			c.Set("count", 42)
			return next(c)
		}
	})

	app.GET("/", func(c *marten.Ctx) error {
		count := c.Get("count").(int)
		return c.JSON(200, map[string]int{"count": count})
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var resp map[string]int
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["count"] != 42 {
		t.Errorf("expected count=42, got %d", resp["count"])
	}
}

func TestContextMethodAndPath(t *testing.T) {
	app := marten.New()
	app.POST("/users/:id", func(c *marten.Ctx) error {
		return c.Text(200, c.Method()+":"+c.Path())
	})

	req := httptest.NewRequest("POST", "/users/123", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "POST:/users/123" {
		t.Errorf("expected POST:/users/123, got %q", rec.Body.String())
	}
}

func TestContextBindValid(t *testing.T) {
	type User struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	app := marten.New()
	app.POST("/users", func(c *marten.Ctx) error {
		var user User
		err := c.BindValid(&user, func() error {
			if user.Name == "" {
				return errors.New("name required")
			}
			if user.Age < 0 {
				return errors.New("age must be positive")
			}
			return nil
		})
		if err != nil {
			return c.BadRequest(err.Error())
		}
		return c.Created(user)
	})

	// Valid input
	body := bytes.NewBufferString(`{"name":"Alice","age":30}`)
	req := httptest.NewRequest("POST", "/users", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Errorf("valid input: expected 201, got %d", rec.Code)
	}

	// Missing name
	body = bytes.NewBufferString(`{"age":30}`)
	req = httptest.NewRequest("POST", "/users", body)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Errorf("missing name: expected 400, got %d", rec.Code)
	}

	// Invalid age
	body = bytes.NewBufferString(`{"name":"Bob","age":-5}`)
	req = httptest.NewRequest("POST", "/users", body)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Errorf("invalid age: expected 400, got %d", rec.Code)
	}
}

func TestContextStatusCode(t *testing.T) {
	app := marten.New()

	var capturedStatus int
	app.Use(func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			err := next(c)
			capturedStatus = c.StatusCode()
			return err
		}
	})

	app.GET("/", func(c *marten.Ctx) error {
		return c.JSON(201, map[string]string{"created": "true"})
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if capturedStatus != 201 {
		t.Errorf("expected StatusCode()=201, got %d", capturedStatus)
	}
}

func TestContextMType(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		return c.OK(marten.M{
			"string": "value",
			"number": 42,
			"bool":   true,
		})
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp["string"] != "value" {
		t.Errorf("expected string=value, got %v", resp["string"])
	}
	if resp["number"].(float64) != 42 {
		t.Errorf("expected number=42, got %v", resp["number"])
	}
	if resp["bool"] != true {
		t.Errorf("expected bool=true, got %v", resp["bool"])
	}
}

func TestContextDoubleWrite(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		c.Text(200, "first")
		c.Text(201, "second") // Should be ignored
		return nil
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	// Body will have both writes but status should be first
	if !strings.Contains(rec.Body.String(), "first") {
		t.Errorf("expected body to contain 'first', got %q", rec.Body.String())
	}
}

func TestContextContext(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		ctx := c.Context()
		if ctx == nil {
			return c.ServerError("context is nil")
		}
		return c.OK(marten.M{"has_context": true})
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
