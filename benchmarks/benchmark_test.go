// Marten Go Framework Benchmarks
// Comparing Marten with Gin, Echo, Chi and Fiber
package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	"github.com/gofiber/fiber/v2"
	"github.com/gomarten/marten"
	"github.com/labstack/echo/v4"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

// ============================================================================
// STATIC ROUTE BENCHMARKS
// ============================================================================

func BenchmarkMarten_StaticRoute(b *testing.B) {
	app := marten.New()
	app.GET("/hello", func(c *marten.Ctx) error {
		return c.Text(200, "Hello, World!")
	})

	req := httptest.NewRequest("GET", "/hello", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkGin_StaticRoute(b *testing.B) {
	app := gin.New()
	app.GET("/hello", func(c *gin.Context) {
		c.String(200, "Hello, World!")
	})

	req := httptest.NewRequest("GET", "/hello", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkEcho_StaticRoute(b *testing.B) {
	app := echo.New()
	app.GET("/hello", func(c echo.Context) error {
		return c.String(200, "Hello, World!")
	})

	req := httptest.NewRequest("GET", "/hello", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkChi_StaticRoute(b *testing.B) {
	app := chi.NewRouter()
	app.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})

	req := httptest.NewRequest("GET", "/hello", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkFiber_StaticRoute(b *testing.B) {
	app := fiber.New()
	app.Get("/hello", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/hello", nil)
		app.Test(req, -1)
	}
}

// ============================================================================
// PARAM ROUTE BENCHMARKS
// ============================================================================

func BenchmarkMarten_ParamRoute(b *testing.B) {
	app := marten.New()
	app.GET("/users/:id", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("id"))
	})

	req := httptest.NewRequest("GET", "/users/123", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkGin_ParamRoute(b *testing.B) {
	app := gin.New()
	app.GET("/users/:id", func(c *gin.Context) {
		c.String(200, c.Param("id"))
	})

	req := httptest.NewRequest("GET", "/users/123", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkEcho_ParamRoute(b *testing.B) {
	app := echo.New()
	app.GET("/users/:id", func(c echo.Context) error {
		return c.String(200, c.Param("id"))
	})

	req := httptest.NewRequest("GET", "/users/123", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkChi_ParamRoute(b *testing.B) {
	app := chi.NewRouter()
	app.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(chi.URLParam(r, "id")))
	})

	req := httptest.NewRequest("GET", "/users/123", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkFiber_ParamRoute(b *testing.B) {
	app := fiber.New()
	app.Get("/users/:id", func(c *fiber.Ctx) error {
		return c.SendString(c.Params("id"))
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/users/123", nil)
		app.Test(req, -1)
	}
}

// ============================================================================
// JSON RESPONSE BENCHMARKS
// ============================================================================

type jsonResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func BenchmarkMarten_JSON(b *testing.B) {
	app := marten.New()
	app.GET("/json", func(c *marten.Ctx) error {
		return c.JSON(200, jsonResponse{Message: "Hello", Status: 200})
	})

	req := httptest.NewRequest("GET", "/json", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkGin_JSON(b *testing.B) {
	app := gin.New()
	app.GET("/json", func(c *gin.Context) {
		c.JSON(200, jsonResponse{Message: "Hello", Status: 200})
	})

	req := httptest.NewRequest("GET", "/json", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkEcho_JSON(b *testing.B) {
	app := echo.New()
	app.GET("/json", func(c echo.Context) error {
		return c.JSON(200, jsonResponse{Message: "Hello", Status: 200})
	})

	req := httptest.NewRequest("GET", "/json", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkChi_JSON(b *testing.B) {
	app := chi.NewRouter()
	app.Get("/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"Hello","status":200}`))
	})

	req := httptest.NewRequest("GET", "/json", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkFiber_JSON(b *testing.B) {
	app := fiber.New()
	app.Get("/json", func(c *fiber.Ctx) error {
		return c.JSON(jsonResponse{Message: "Hello", Status: 200})
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/json", nil)
		app.Test(req, -1)
	}
}

// ============================================================================
// JSON BINDING BENCHMARKS
// ============================================================================

type bindRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func BenchmarkMarten_JSONBind(b *testing.B) {
	app := marten.New()
	app.POST("/bind", func(c *marten.Ctx) error {
		var req bindRequest
		if err := c.Bind(&req); err != nil {
			return err
		}
		return c.JSON(200, req)
	})

	body := `{"name":"John","email":"john@example.com"}`
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/bind", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkGin_JSONBind(b *testing.B) {
	app := gin.New()
	app.POST("/bind", func(c *gin.Context) {
		var req bindRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, req)
	})

	body := `{"name":"John","email":"john@example.com"}`
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/bind", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkEcho_JSONBind(b *testing.B) {
	app := echo.New()
	app.POST("/bind", func(c echo.Context) error {
		var req bindRequest
		if err := c.Bind(&req); err != nil {
			return err
		}
		return c.JSON(200, req)
	})

	body := `{"name":"John","email":"john@example.com"}`
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/bind", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkChi_JSONBind(b *testing.B) {
	app := chi.NewRouter()
	app.Post("/bind", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	})

	body := `{"name":"John","email":"john@example.com"}`
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/bind", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkFiber_JSONBind(b *testing.B) {
	app := fiber.New()
	app.Post("/bind", func(c *fiber.Ctx) error {
		var req bindRequest
		if err := c.BodyParser(&req); err != nil {
			return err
		}
		return c.JSON(req)
	})

	body := `{"name":"John","email":"john@example.com"}`
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/bind", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		app.Test(req, -1)
	}
}

// ============================================================================
// MULTI-PARAM ROUTE BENCHMARKS
// ============================================================================

func BenchmarkMarten_MultiParam(b *testing.B) {
	app := marten.New()
	app.GET("/users/:userId/posts/:postId/comments/:commentId", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("userId")+c.Param("postId")+c.Param("commentId"))
	})

	req := httptest.NewRequest("GET", "/users/1/posts/2/comments/3", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkGin_MultiParam(b *testing.B) {
	app := gin.New()
	app.GET("/users/:userId/posts/:postId/comments/:commentId", func(c *gin.Context) {
		c.String(200, c.Param("userId")+c.Param("postId")+c.Param("commentId"))
	})

	req := httptest.NewRequest("GET", "/users/1/posts/2/comments/3", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkEcho_MultiParam(b *testing.B) {
	app := echo.New()
	app.GET("/users/:userId/posts/:postId/comments/:commentId", func(c echo.Context) error {
		return c.String(200, c.Param("userId")+c.Param("postId")+c.Param("commentId"))
	})

	req := httptest.NewRequest("GET", "/users/1/posts/2/comments/3", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkChi_MultiParam(b *testing.B) {
	app := chi.NewRouter()
	app.Get("/users/{userId}/posts/{postId}/comments/{commentId}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(chi.URLParam(r, "userId") + chi.URLParam(r, "postId") + chi.URLParam(r, "commentId")))
	})

	req := httptest.NewRequest("GET", "/users/1/posts/2/comments/3", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkFiber_MultiParam(b *testing.B) {
	app := fiber.New()
	app.Get("/users/:userId/posts/:postId/comments/:commentId", func(c *fiber.Ctx) error {
		return c.SendString(c.Params("userId") + c.Params("postId") + c.Params("commentId"))
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/users/1/posts/2/comments/3", nil)
		app.Test(req, -1)
	}
}

// ============================================================================
// QUERY PARAMS BENCHMARKS
// ============================================================================

func BenchmarkMarten_QueryParams(b *testing.B) {
	app := marten.New()
	app.GET("/search", func(c *marten.Ctx) error {
		q := c.Query("q")
		page := c.QueryInt("page")
		limit := c.QueryInt("limit")
		return c.JSON(200, marten.M{"q": q, "page": page, "limit": limit})
	})

	req := httptest.NewRequest("GET", "/search?q=golang&page=1&limit=10", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkGin_QueryParams(b *testing.B) {
	app := gin.New()
	app.GET("/search", func(c *gin.Context) {
		q := c.Query("q")
		page := c.Query("page")
		limit := c.Query("limit")
		c.JSON(200, gin.H{"q": q, "page": page, "limit": limit})
	})

	req := httptest.NewRequest("GET", "/search?q=golang&page=1&limit=10", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkEcho_QueryParams(b *testing.B) {
	app := echo.New()
	app.GET("/search", func(c echo.Context) error {
		q := c.QueryParam("q")
		page := c.QueryParam("page")
		limit := c.QueryParam("limit")
		return c.JSON(200, map[string]string{"q": q, "page": page, "limit": limit})
	})

	req := httptest.NewRequest("GET", "/search?q=golang&page=1&limit=10", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

// ============================================================================
// LARGE JSON RESPONSE BENCHMARKS
// ============================================================================

type largeUser struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	Email     string   `json:"email"`
	Age       int      `json:"age"`
	Address   string   `json:"address"`
	City      string   `json:"city"`
	Country   string   `json:"country"`
	Phone     string   `json:"phone"`
	Company   string   `json:"company"`
	Tags      []string `json:"tags"`
	Active    bool     `json:"active"`
	CreatedAt string   `json:"created_at"`
}

var largeUserData = largeUser{
	ID:        1,
	Name:      "John Doe",
	Email:     "john@example.com",
	Age:       30,
	Address:   "123 Main St",
	City:      "New York",
	Country:   "USA",
	Phone:     "+1234567890",
	Company:   "Acme Corp",
	Tags:      []string{"developer", "golang", "backend", "api"},
	Active:    true,
	CreatedAt: "2024-01-01T00:00:00Z",
}

func BenchmarkMarten_LargeJSON(b *testing.B) {
	app := marten.New()
	app.GET("/user", func(c *marten.Ctx) error {
		return c.JSON(200, largeUserData)
	})

	req := httptest.NewRequest("GET", "/user", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkGin_LargeJSON(b *testing.B) {
	app := gin.New()
	app.GET("/user", func(c *gin.Context) {
		c.JSON(200, largeUserData)
	})

	req := httptest.NewRequest("GET", "/user", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkEcho_LargeJSON(b *testing.B) {
	app := echo.New()
	app.GET("/user", func(c echo.Context) error {
		return c.JSON(200, largeUserData)
	})

	req := httptest.NewRequest("GET", "/user", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

// ============================================================================
// ROUTE GROUP BENCHMARKS
// ============================================================================

func BenchmarkMarten_RouteGroup(b *testing.B) {
	app := marten.New()
	api := app.Group("/api/v1")
	api.GET("/users/:id", func(c *marten.Ctx) error {
		return c.JSON(200, marten.M{"id": c.Param("id")})
	})

	req := httptest.NewRequest("GET", "/api/v1/users/123", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkGin_RouteGroup(b *testing.B) {
	app := gin.New()
	api := app.Group("/api/v1")
	api.GET("/users/:id", func(c *gin.Context) {
		c.JSON(200, gin.H{"id": c.Param("id")})
	})

	req := httptest.NewRequest("GET", "/api/v1/users/123", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkEcho_RouteGroup(b *testing.B) {
	app := echo.New()
	api := app.Group("/api/v1")
	api.GET("/users/:id", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"id": c.Param("id")})
	})

	req := httptest.NewRequest("GET", "/api/v1/users/123", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

// ============================================================================
// WILDCARD ROUTE BENCHMARKS
// ============================================================================

func BenchmarkMarten_WildcardRoute(b *testing.B) {
	app := marten.New()
	app.GET("/files/*filepath", func(c *marten.Ctx) error {
		return c.Text(200, c.Param("filepath"))
	})

	req := httptest.NewRequest("GET", "/files/documents/report.pdf", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkGin_WildcardRoute(b *testing.B) {
	app := gin.New()
	app.GET("/files/*filepath", func(c *gin.Context) {
		c.String(200, c.Param("filepath"))
	})

	req := httptest.NewRequest("GET", "/files/documents/report.pdf", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

func BenchmarkEcho_WildcardRoute(b *testing.B) {
	app := echo.New()
	app.GET("/files/*", func(c echo.Context) error {
		return c.String(200, c.Param("*"))
	})

	req := httptest.NewRequest("GET", "/files/documents/report.pdf", nil)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		app.ServeHTTP(w, req)
	}
}

// ============================================================================
// PARALLEL BENCHMARKS
// ============================================================================

func BenchmarkMarten_StaticRoute_Parallel(b *testing.B) {
	app := marten.New()
	app.GET("/hello", func(c *marten.Ctx) error {
		return c.Text(200, "Hello, World!")
	})

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/hello", nil)
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)
		}
	})
}

func BenchmarkGin_StaticRoute_Parallel(b *testing.B) {
	app := gin.New()
	app.GET("/hello", func(c *gin.Context) {
		c.String(200, "Hello, World!")
	})

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/hello", nil)
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)
		}
	})
}

func BenchmarkEcho_StaticRoute_Parallel(b *testing.B) {
	app := echo.New()
	app.GET("/hello", func(c echo.Context) error {
		return c.String(200, "Hello, World!")
	})

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/hello", nil)
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)
		}
	})
}
