package marten

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
)

// Ctx wraps a request with helpers for clean handler code.
type Ctx struct {
	Request    *http.Request
	Writer     http.ResponseWriter
	params     map[string]string
	store      map[string]any
	written    bool
	statusCode int
	requestID  string
}

// Param returns a path parameter by name.
func (c *Ctx) Param(name string) string {
	return c.params[name]
}

// ParamInt returns a path parameter as int (0 if invalid).
func (c *Ctx) ParamInt(name string) int {
	v, _ := strconv.Atoi(c.params[name])
	return v
}

// ParamInt64 returns a path parameter as int64 (0 if invalid).
func (c *Ctx) ParamInt64(name string) int64 {
	v, _ := strconv.ParseInt(c.params[name], 10, 64)
	return v
}

// Query returns a query parameter by name.
func (c *Ctx) Query(name string) string {
	if c.Request.URL == nil {
		return ""
	}
	return c.Request.URL.Query().Get(name)
}

// QueryInt returns a query parameter as int (0 if invalid).
func (c *Ctx) QueryInt(name string) int {
	v, _ := strconv.Atoi(c.Query(name))
	return v
}

// QueryInt64 returns a query parameter as int64 (0 if invalid).
func (c *Ctx) QueryInt64(name string) int64 {
	v, _ := strconv.ParseInt(c.Query(name), 10, 64)
	return v
}

// QueryBool returns a query parameter as bool (false if invalid).
func (c *Ctx) QueryBool(name string) bool {
	v, _ := strconv.ParseBool(c.Query(name))
	return v
}

// QueryDefault returns a query parameter or default if empty.
func (c *Ctx) QueryDefault(name, def string) string {
	if v := c.Query(name); v != "" {
		return v
	}
	return def
}

// QueryValues returns all values for a query parameter.
func (c *Ctx) QueryValues(name string) []string {
	if c.Request.URL == nil {
		return nil
	}
	return c.Request.URL.Query()[name]
}

// Status sets the response status code.
func (c *Ctx) Status(code int) *Ctx {
	if !c.written {
		c.Writer.WriteHeader(code)
		c.written = true
		c.statusCode = code
	}
	return c
}

// StatusCode returns the response status code (0 if not yet written).
func (c *Ctx) StatusCode() int {
	return c.statusCode
}

// Text writes a plain text response.
func (c *Ctx) Text(code int, text string) error {
	if !c.written {
		c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		c.Writer.WriteHeader(code)
		c.written = true
		c.statusCode = code
	}
	_, err := c.Writer.Write([]byte(text))
	return err
}

// JSON writes a JSON response.
func (c *Ctx) JSON(code int, v any) error {
	if !c.written {
		c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		c.Writer.WriteHeader(code)
		c.written = true
		c.statusCode = code
	}
	return json.NewEncoder(c.Writer).Encode(v)
}

// OK sends a 200 JSON response.
func (c *Ctx) OK(v any) error {
	return c.JSON(http.StatusOK, v)
}

// Created sends a 201 JSON response.
func (c *Ctx) Created(v any) error {
	return c.JSON(http.StatusCreated, v)
}

// NoContent sends a 204 response.
func (c *Ctx) NoContent() error {
	c.Status(http.StatusNoContent)
	return nil
}

// BadRequest sends a 400 JSON error response.
func (c *Ctx) BadRequest(message string) error {
	return c.JSON(http.StatusBadRequest, E(message))
}

// Unauthorized sends a 401 JSON error response.
func (c *Ctx) Unauthorized(message string) error {
	return c.JSON(http.StatusUnauthorized, E(message))
}

// Forbidden sends a 403 JSON error response.
func (c *Ctx) Forbidden(message string) error {
	return c.JSON(http.StatusForbidden, E(message))
}

// NotFound sends a 404 JSON error response.
func (c *Ctx) NotFound(message string) error {
	return c.JSON(http.StatusNotFound, E(message))
}

// ServerError sends a 500 JSON error response.
func (c *Ctx) ServerError(message string) error {
	return c.JSON(http.StatusInternalServerError, E(message))
}

// E creates a simple error response map.
func E(message string) map[string]string {
	return map[string]string{"error": message}
}

// M is a shorthand for map[string]any.
type M map[string]any

// Redirect sends a redirect response.
func (c *Ctx) Redirect(code int, url string) error {
	c.Writer.Header().Set("Location", url)
	c.Status(code)
	return nil
}

// Context returns the request's context.
func (c *Ctx) Context() context.Context {
	if c.Request == nil {
		return context.Background()
	}
	return c.Request.Context()
}

// Bind decodes JSON request body into v.
func (c *Ctx) Bind(v any) error {
	if c.Request.Body == nil {
		return &BindError{Message: "empty request body"}
	}
	if err := json.NewDecoder(c.Request.Body).Decode(v); err != nil {
		return &BindError{Message: "invalid JSON: " + err.Error()}
	}
	return nil
}

// BindValid decodes JSON and validates using the provided function.
func (c *Ctx) BindValid(v any, validate func() error) error {
	if err := c.Bind(v); err != nil {
		return err
	}
	return validate()
}

// BindError represents a binding error.
type BindError struct {
	Message string
}

func (e *BindError) Error() string {
	return e.Message
}

// RequestID returns a unique request identifier.
func (c *Ctx) RequestID() string {
	if c.requestID == "" {
		if id := c.Request.Header.Get("X-Request-ID"); id != "" {
			c.requestID = id
		} else {
			b := make([]byte, 8)
			_, _ = rand.Read(b)
			c.requestID = hex.EncodeToString(b)
		}
	}
	return c.requestID
}

// ClientIP extracts the client IP intelligently.
func (c *Ctx) ClientIP() string {
	if c.Request == nil {
		return ""
	}
	if xff := c.Request.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.Index(xff, ","); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if xri := c.Request.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	addr := c.Request.RemoteAddr
	if strings.HasPrefix(addr, "[") {
		if i := strings.LastIndex(addr, "]"); i > 0 {
			return addr[1:i]
		}
	}
	if i := strings.LastIndex(addr, ":"); i > 0 {
		return addr[:i]
	}
	return addr
}

// Bearer extracts the Bearer token from Authorization header.
func (c *Ctx) Bearer() string {
	auth := c.Request.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return auth[7:]
	}
	return ""
}

// IsJSON returns true if Content-Type is application/json.
func (c *Ctx) IsJSON() bool {
	return strings.HasPrefix(c.Request.Header.Get("Content-Type"), "application/json")
}

// IsAJAX returns true if X-Requested-With is XMLHttpRequest.
func (c *Ctx) IsAJAX() bool {
	return c.Request.Header.Get("X-Requested-With") == "XMLHttpRequest"
}

// Method returns the request method.
func (c *Ctx) Method() string {
	return c.Request.Method
}

// Path returns the request path.
func (c *Ctx) Path() string {
	return c.Request.URL.Path
}

// Set stores a value in the request context.
func (c *Ctx) Set(key string, value any) {
	if c.store == nil {
		c.store = make(map[string]any)
	}
	c.store[key] = value
}

// Get retrieves a value from the request context.
func (c *Ctx) Get(key string) any {
	if c.store == nil {
		return nil
	}
	return c.store[key]
}

// GetString retrieves a string value from the request context.
func (c *Ctx) GetString(key string) string {
	if v, ok := c.Get(key).(string); ok {
		return v
	}
	return ""
}

// GetInt retrieves an int value from the request context.
func (c *Ctx) GetInt(key string) int {
	if v, ok := c.Get(key).(int); ok {
		return v
	}
	return 0
}

// GetBool retrieves a bool value from the request context.
func (c *Ctx) GetBool(key string) bool {
	if v, ok := c.Get(key).(bool); ok {
		return v
	}
	return false
}

// Cookie returns a cookie value by name.
func (c *Ctx) Cookie(name string) string {
	cookie, err := c.Request.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// SetCookie sets a response cookie.
func (c *Ctx) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.Writer, cookie)
}

// FormValue returns a form value by name.
func (c *Ctx) FormValue(name string) string {
	return c.Request.FormValue(name)
}

// File returns a file from multipart form.
func (c *Ctx) File(name string) (*multipart.FileHeader, error) {
	_, fh, err := c.Request.FormFile(name)
	return fh, err
}

// Header sets a response header.
func (c *Ctx) Header(key, value string) *Ctx {
	c.Writer.Header().Set(key, value)
	return c
}

// SetParam sets a path parameter (used internally by router).
func (c *Ctx) SetParam(key, value string) {
	c.params[key] = value
}

// Reset clears the context for reuse.
func (c *Ctx) Reset(w http.ResponseWriter, r *http.Request) {
	c.Writer = w
	c.Request = r
	c.written = false
	c.statusCode = 0
	c.requestID = ""
	for k := range c.params {
		delete(c.params, k)
	}
	for k := range c.store {
		delete(c.store, k)
	}
}
