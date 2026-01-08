// Basic example - Hello World with Marten
package main

import (
	"log"

	"github.com/gomarten/marten"
)

func main() {
	app := marten.New()

	// Simple text response
	app.GET("/", func(c *marten.Ctx) error {
		return c.Text(200, "Hello, Marten!")
	})

	// JSON response
	app.GET("/json", func(c *marten.Ctx) error {
		return c.OK(marten.M{
			"message": "Hello, JSON!",
			"status":  "success",
		})
	})

	// Path parameters
	app.GET("/users/:id", func(c *marten.Ctx) error {
		id := c.Param("id")
		return c.OK(marten.M{"user_id": id})
	})

	// Query parameters
	app.GET("/search", func(c *marten.Ctx) error {
		q := c.Query("q")
		page := c.QueryInt("page")
		if page == 0 {
			page = 1
		}
		return c.OK(marten.M{
			"query": q,
			"page":  page,
		})
	})

	log.Println("Server running on http://localhost:3000")
	app.Run(":3000")
}
