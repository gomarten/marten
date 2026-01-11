// CRUD API example - RESTful API with Marten
package main

import (
	"log"
	"sync"
	"time"

	"github.com/gomarten/marten"
	"github.com/gomarten/marten/middleware"
)

// User represents a user in our system
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// In-memory store
var (
	users   = make(map[string]User)
	usersMu sync.RWMutex
	nextID  = 1
)

func main() {
	app := marten.New()

	// Global middleware
	app.Use(
		middleware.RequestID,
		middleware.Logger,
		middleware.Recover,
		middleware.CORS(middleware.DefaultCORSConfig()),
	)

	// Health check
	app.GET("/health", func(c *marten.Ctx) error {
		return c.OK(marten.M{"status": "healthy"})
	})

	// API routes
	api := app.Group("/api/v1")
	{
		// Users CRUD
		api.GET("/users", listUsers)
		api.GET("/users/:id", getUser)
		api.POST("/users", createUser)
		api.PUT("/users/:id", updateUser)
		api.DELETE("/users/:id", deleteUser)
	}

	log.Println("CRUD API running on http://localhost:3000")
	app.RunGraceful(":3000", 10*time.Second)
}

func listUsers(c *marten.Ctx) error {
	usersMu.RLock()
	defer usersMu.RUnlock()

	list := make([]User, 0, len(users))
	for _, u := range users {
		list = append(list, u)
	}

	return c.OK(marten.M{
		"users": list,
		"total": len(list),
	})
}

func getUser(c *marten.Ctx) error {
	id := c.Param("id")

	usersMu.RLock()
	user, exists := users[id]
	usersMu.RUnlock()

	if !exists {
		return c.NotFound("user not found")
	}

	return c.OK(user)
}

func createUser(c *marten.Ctx) error {
	var input struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := c.BindValid(&input, func() error {
		if input.Name == "" {
			return &marten.BindError{Message: "name is required"}
		}
		if input.Email == "" {
			return &marten.BindError{Message: "email is required"}
		}
		return nil
	}); err != nil {
		return c.BadRequest(err.Error())
	}

	usersMu.Lock()
	id := string(rune('0' + nextID))
	nextID++
	user := User{
		ID:        id,
		Name:      input.Name,
		Email:     input.Email,
		CreatedAt: time.Now(),
	}
	users[id] = user
	usersMu.Unlock()

	return c.Created(user)
}

func updateUser(c *marten.Ctx) error {
	id := c.Param("id")

	usersMu.RLock()
	user, exists := users[id]
	usersMu.RUnlock()

	if !exists {
		return c.NotFound("user not found")
	}

	var input struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := c.Bind(&input); err != nil {
		return c.BadRequest(err.Error())
	}

	if input.Name != "" {
		user.Name = input.Name
	}
	if input.Email != "" {
		user.Email = input.Email
	}

	usersMu.Lock()
	users[id] = user
	usersMu.Unlock()

	return c.OK(user)
}

func deleteUser(c *marten.Ctx) error {
	id := c.Param("id")

	usersMu.Lock()
	_, exists := users[id]
	if exists {
		delete(users, id)
	}
	usersMu.Unlock()

	if !exists {
		return c.NotFound("user not found")
	}

	return c.NoContent()
}
