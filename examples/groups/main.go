// Route Groups example - Organizing routes with groups and middleware
package main

import (
	"log"

	"github.com/gomarten/marten"
	"github.com/gomarten/marten/middleware"
)

func main() {
	app := marten.New()

	app.Use(middleware.Logger, middleware.Recover)

	// Public routes
	app.GET("/", func(c *marten.Ctx) error {
		return c.OK(marten.M{"message": "Welcome to the API"})
	})

	app.GET("/health", func(c *marten.Ctx) error {
		return c.OK(marten.M{"status": "healthy"})
	})

	// API v1
	v1 := app.Group("/api/v1")
	{
		// Users
		users := v1.Group("/users")
		users.GET("", listUsersV1)
		users.GET("/:id", getUserV1)
		users.POST("", createUserV1)

		// Posts
		posts := v1.Group("/posts")
		posts.GET("", listPostsV1)
		posts.GET("/:id", getPostV1)

		// Nested: User's posts
		users.GET("/:id/posts", getUserPostsV1)
	}

	// API v2 with different middleware
	v2 := app.Group("/api/v2")
	v2.Use(V2Middleware)
	{
		users := v2.Group("/users")
		users.GET("", listUsersV2)
		users.GET("/:id", getUserV2)
	}

	// Admin routes with auth
	admin := app.Group("/admin")
	admin.Use(middleware.BasicAuthSimple("admin", "secret"))
	{
		admin.GET("/dashboard", adminDashboard)
		admin.GET("/stats", adminStats)

		// Nested admin groups
		adminUsers := admin.Group("/users")
		adminUsers.GET("", adminListUsers)
		adminUsers.DELETE("/:id", adminDeleteUser)
	}

	// Webhooks with specific middleware
	webhooks := app.Group("/webhooks")
	webhooks.Use(WebhookAuthMiddleware)
	{
		webhooks.POST("/github", githubWebhook)
		webhooks.POST("/stripe", stripeWebhook)
	}

	log.Println("Groups example running on http://localhost:3000")
	log.Println("")
	log.Println("Registered routes:")
	for _, r := range app.Routes() {
		log.Printf("  %-6s %s", r.Method, r.Path)
	}

	app.Run(":3000")
}

// --- V1 Handlers ---

func listUsersV1(c *marten.Ctx) error {
	return c.OK(marten.M{
		"version": "v1",
		"users":   []marten.M{{"id": "1", "name": "Alice"}},
	})
}

func getUserV1(c *marten.Ctx) error {
	return c.OK(marten.M{
		"version": "v1",
		"user":    marten.M{"id": c.Param("id"), "name": "Alice"},
	})
}

func createUserV1(c *marten.Ctx) error {
	return c.Created(marten.M{"version": "v1", "message": "user created"})
}

func listPostsV1(c *marten.Ctx) error {
	return c.OK(marten.M{
		"version": "v1",
		"posts":   []marten.M{{"id": "1", "title": "Hello World"}},
	})
}

func getPostV1(c *marten.Ctx) error {
	return c.OK(marten.M{
		"version": "v1",
		"post":    marten.M{"id": c.Param("id"), "title": "Hello World"},
	})
}

func getUserPostsV1(c *marten.Ctx) error {
	return c.OK(marten.M{
		"version": "v1",
		"user_id": c.Param("id"),
		"posts":   []marten.M{{"id": "1", "title": "My Post"}},
	})
}

// --- V2 Handlers ---

func listUsersV2(c *marten.Ctx) error {
	return c.OK(marten.M{
		"version": "v2",
		"data": marten.M{
			"users": []marten.M{{"id": "1", "name": "Alice", "email": "alice@example.com"}},
			"total": 1,
		},
	})
}

func getUserV2(c *marten.Ctx) error {
	return c.OK(marten.M{
		"version": "v2",
		"data": marten.M{
			"id":    c.Param("id"),
			"name":  "Alice",
			"email": "alice@example.com",
		},
	})
}

// --- Admin Handlers ---

func adminDashboard(c *marten.Ctx) error {
	return c.OK(marten.M{
		"message": "Admin Dashboard",
		"user":    c.GetString("user"),
	})
}

func adminStats(c *marten.Ctx) error {
	return c.OK(marten.M{
		"total_users": 100,
		"total_posts": 500,
		"active":      42,
	})
}

func adminListUsers(c *marten.Ctx) error {
	return c.OK(marten.M{
		"users": []marten.M{
			{"id": "1", "name": "Alice", "role": "user"},
			{"id": "2", "name": "Bob", "role": "admin"},
		},
	})
}

func adminDeleteUser(c *marten.Ctx) error {
	return c.OK(marten.M{"deleted": c.Param("id")})
}

// --- Webhook Handlers ---

func githubWebhook(c *marten.Ctx) error {
	return c.OK(marten.M{"received": "github"})
}

func stripeWebhook(c *marten.Ctx) error {
	return c.OK(marten.M{"received": "stripe"})
}

// --- Middleware ---

func V2Middleware(next marten.Handler) marten.Handler {
	return func(c *marten.Ctx) error {
		c.Header("X-API-Version", "2.0")
		return next(c)
	}
}

func WebhookAuthMiddleware(next marten.Handler) marten.Handler {
	return func(c *marten.Ctx) error {
		secret := c.Request.Header.Get("X-Webhook-Secret")
		if secret != "webhook-secret-123" {
			return c.Unauthorized("invalid webhook secret")
		}
		return next(c)
	}
}
