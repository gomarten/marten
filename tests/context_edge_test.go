package tests

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gomarten/marten"
)

// --- Bind Edge Cases ---

func TestBindEmptyBody(t *testing.T) {
	app := marten.New()
	app.POST("/", func(c *marten.Ctx) error {
		var data map[string]string
		if err := c.Bind(&data); err != nil {
			return c.BadRequest(err.Error())
		}
		return c.OK(data)
	})

	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 400 {
		t.Errorf("expected 400 for empty body, got %d", rec.Code)
	}
}

func TestBindInvalidJSON(t *testing.T) {
	app := marten.New()
	app.POST("/", func(c *marten.Ctx) error {
		var data map[string]string
		if err := c.Bind(&data); err != nil {
			return c.BadRequest(err.Error())
		}
		return c.OK(data)
	})

	tests := []struct {
		name string
		body string
	}{
		{"malformed", `{"name": "test"`},
		{"wrong type", `["array"]`},
		{"invalid syntax", `{name: test}`},
		{"trailing comma", `{"name": "test",}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Code != 400 {
				t.Errorf("%s: expected 400, got %d", tt.name, rec.Code)
			}
		})
	}
}

func TestBindValidWithError(t *testing.T) {
	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	app := marten.New()
	app.POST("/", func(c *marten.Ctx) error {
		var user User
		err := c.BindValid(&user, func() error {
			if user.Name == "" {
				return &marten.BindError{Message: "name required"}
			}
			if !strings.Contains(user.Email, "@") {
				return &marten.BindError{Message: "invalid email"}
			}
			return nil
		})
		if err != nil {
			return c.BadRequest(err.Error())
		}
		return c.Created(user)
	})

	tests := []struct {
		name     string
		body     string
		expected int
	}{
		{"valid", `{"name":"Alice","email":"alice@example.com"}`, 201},
		{"missing name", `{"email":"alice@example.com"}`, 400},
		{"invalid email", `{"name":"Alice","email":"invalid"}`, 400},
		{"invalid json", `{invalid}`, 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Code != tt.expected {
				t.Errorf("%s: expected %d, got %d: %s", tt.name, tt.expected, rec.Code, rec.Body.String())
			}
		})
	}
}

// --- Query Edge Cases ---

func TestQueryInt64(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		id := c.QueryInt64("id")
		return c.JSON(200, map[string]int64{"id": id})
	})

	req := httptest.NewRequest("GET", "/?id=9223372036854775807", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "9223372036854775807") {
		t.Errorf("expected max int64, got %s", rec.Body.String())
	}
}

func TestQueryBool(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		active := c.QueryBool("active")
		return c.JSON(200, map[string]bool{"active": active})
	})

	tests := []struct {
		query    string
		expected string
	}{
		{"?active=true", "true"},
		{"?active=false", "false"},
		{"?active=1", "true"},
		{"?active=0", "false"},
		{"?active=invalid", "false"},
		{"", "false"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/"+tt.query, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if !strings.Contains(rec.Body.String(), tt.expected) {
			t.Errorf("query %q: expected %s, got %s", tt.query, tt.expected, rec.Body.String())
		}
	}
}

func TestQueryValues(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		tags := c.QueryValues("tag")
		return c.JSON(200, map[string][]string{"tags": tags})
	})

	req := httptest.NewRequest("GET", "/?tag=a&tag=b&tag=c", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "a") || !strings.Contains(body, "b") || !strings.Contains(body, "c") {
		t.Errorf("expected all tags, got %s", body)
	}
}

func TestQueryEmptyValues(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		tags := c.QueryValues("missing")
		if tags == nil {
			return c.Text(200, "nil")
		}
		return c.Text(200, "not nil")
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "nil" {
		t.Errorf("expected nil for missing query values, got %s", rec.Body.String())
	}
}

// --- Store Edge Cases ---

func TestStoreGetInt(t *testing.T) {
	app := marten.New()
	app.Use(func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			c.Set("count", 42)
			c.Set("invalid", "not an int")
			return next(c)
		}
	})
	app.GET("/", func(c *marten.Ctx) error {
		count := c.GetInt("count")
		invalid := c.GetInt("invalid")
		missing := c.GetInt("missing")
		return c.JSON(200, map[string]int{
			"count":   count,
			"invalid": invalid,
			"missing": missing,
		})
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `"count":42`) {
		t.Errorf("expected count=42, got %s", body)
	}
	if !strings.Contains(body, `"invalid":0`) {
		t.Errorf("expected invalid=0, got %s", body)
	}
	if !strings.Contains(body, `"missing":0`) {
		t.Errorf("expected missing=0, got %s", body)
	}
}

func TestStoreGetBool(t *testing.T) {
	app := marten.New()
	app.Use(func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			c.Set("active", true)
			c.Set("invalid", "not a bool")
			return next(c)
		}
	})
	app.GET("/", func(c *marten.Ctx) error {
		active := c.GetBool("active")
		invalid := c.GetBool("invalid")
		missing := c.GetBool("missing")
		return c.JSON(200, map[string]bool{
			"active":  active,
			"invalid": invalid,
			"missing": missing,
		})
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `"active":true`) {
		t.Errorf("expected active=true, got %s", body)
	}
	if !strings.Contains(body, `"invalid":false`) {
		t.Errorf("expected invalid=false, got %s", body)
	}
}

// --- Cookie Edge Cases ---

func TestCookie(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		session := c.Cookie("session")
		missing := c.Cookie("missing")
		return c.Text(200, session+":"+missing)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "abc123"})
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "abc123:" {
		t.Errorf("expected 'abc123:', got %q", rec.Body.String())
	}
}

func TestSetCookie(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		c.SetCookie(&http.Cookie{
			Name:     "session",
			Value:    "xyz789",
			Path:     "/",
			HttpOnly: true,
		})
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Name != "session" || cookies[0].Value != "xyz789" {
		t.Errorf("unexpected cookie: %+v", cookies[0])
	}
}

// --- ClientIP Edge Cases ---

func TestClientIPIPv6(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, c.ClientIP())
	})

	// Test IPv6 with port
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "[::1]:8080"
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "::1" {
		t.Errorf("expected '::1', got %q", rec.Body.String())
	}
}

func TestClientIPNoPort(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, c.ClientIP())
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1"
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Should handle address without port
	if rec.Body.String() != "192.168.1.1" {
		t.Errorf("expected '192.168.1.1', got %q", rec.Body.String())
	}
}

// --- Response Edge Cases ---

func TestJSONNilValue(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		return c.JSON(200, nil)
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != "null" {
		t.Errorf("expected 'null', got %q", rec.Body.String())
	}
}

func TestTextEmptyString(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "")
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "" {
		t.Errorf("expected empty body, got %q", rec.Body.String())
	}
}

func TestRedirectCodes(t *testing.T) {
	tests := []struct {
		code int
		url  string
	}{
		{301, "/permanent"},
		{302, "/found"},
		{303, "/see-other"},
		{307, "/temporary"},
		{308, "/permanent-redirect"},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.code), func(t *testing.T) {
			app := marten.New()
			app.GET("/", func(c *marten.Ctx) error {
				return c.Redirect(tt.code, tt.url)
			})

			req := httptest.NewRequest("GET", "/", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			if rec.Code != tt.code {
				t.Errorf("expected %d, got %d", tt.code, rec.Code)
			}
			if rec.Header().Get("Location") != tt.url {
				t.Errorf("expected Location=%s, got %s", tt.url, rec.Header().Get("Location"))
			}
		})
	}
}

// --- FormValue Edge Cases ---

func TestFormValue(t *testing.T) {
	app := marten.New()
	app.POST("/", func(c *marten.Ctx) error {
		name := c.FormValue("name")
		missing := c.FormValue("missing")
		return c.Text(200, name+":"+missing)
	})

	req := httptest.NewRequest("POST", "/", strings.NewReader("name=Alice"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "Alice:" {
		t.Errorf("expected 'Alice:', got %q", rec.Body.String())
	}
}

// --- Header Edge Cases ---

func TestHeaderChaining(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		c.Header("X-Custom-1", "value1").
			Header("X-Custom-2", "value2").
			Header("X-Custom-3", "value3")
		return c.Text(200, "ok")
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Header().Get("X-Custom-1") != "value1" {
		t.Error("missing X-Custom-1")
	}
	if rec.Header().Get("X-Custom-2") != "value2" {
		t.Error("missing X-Custom-2")
	}
	if rec.Header().Get("X-Custom-3") != "value3" {
		t.Error("missing X-Custom-3")
	}
}

// --- Context Edge Cases ---

func TestContextNilRequest(t *testing.T) {
	// Test that Context() handles nil request gracefully
	ctx := &marten.Ctx{}
	c := ctx.Context()
	if c == nil {
		t.Error("Context() should return non-nil context even with nil request")
	}
}

// --- Param Edge Cases ---

func TestParamSpecialCharacters(t *testing.T) {
	app := marten.New()
	app.GET("/files/:name", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("name"))
	})

	tests := []struct {
		path     string
		expected string
	}{
		{"/files/test.txt", "test.txt"},
		{"/files/my-file", "my-file"},
		{"/files/file_name", "file_name"},
		{"/files/file%20name", "file name"}, // URL decoded by Go
		{"/files/123", "123"},
		{"/files/a.b.c", "a.b.c"},
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

func TestParamMissing(t *testing.T) {
	app := marten.New()
	app.GET("/users/:id", func(c *marten.Ctx) error {
		id := c.Param("id")
		missing := c.Param("missing")
		return c.Text(200, id+":"+missing)
	})

	req := httptest.NewRequest("GET", "/users/123", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Body.String() != "123:" {
		t.Errorf("expected '123:', got %q", rec.Body.String())
	}
}

// --- M Type Edge Cases ---

func TestMTypeNested(t *testing.T) {
	app := marten.New()
	app.GET("/", func(c *marten.Ctx) error {
		return c.OK(marten.M{
			"user": marten.M{
				"name": "Alice",
				"meta": marten.M{
					"active": true,
				},
			},
			"count": 42,
		})
	})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "Alice") {
		t.Errorf("expected nested name, got %s", body)
	}
	if !strings.Contains(body, "active") {
		t.Errorf("expected nested active, got %s", body)
	}
}
