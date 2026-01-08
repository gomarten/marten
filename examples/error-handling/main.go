// Error Handling example - Custom error handling and validation
package main

import (
	"errors"
	"log"

	"github.com/gomarten/marten"
	"github.com/gomarten/marten/middleware"
)

// Custom error types
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	return e.Resource + " not found: " + e.ID
}

type ForbiddenError struct {
	Message string
}

func (e *ForbiddenError) Error() string {
	return e.Message
}

func main() {
	app := marten.New()

	app.Use(middleware.Logger, middleware.Recover)

	// Custom error handler
	app.OnError(customErrorHandler)

	// Routes that demonstrate error handling
	app.GET("/users/:id", getUser)
	app.POST("/users", createUser)
	app.DELETE("/users/:id", deleteUser)

	// Route that panics (recovered by middleware)
	app.GET("/panic", func(c *marten.Ctx) error {
		panic("something went terribly wrong!")
	})

	// Route with validation
	app.POST("/validate", validateInput)

	log.Println("Error handling example running on http://localhost:3000")
	log.Println("")
	log.Println("Try these:")
	log.Println("  GET  /users/123      - returns user")
	log.Println("  GET  /users/999      - returns 404")
	log.Println("  POST /users          - with invalid JSON")
	log.Println("  DELETE /users/123    - returns 403")
	log.Println("  GET  /panic          - triggers panic recovery")
	log.Println("  POST /validate       - validation example")

	app.Run(":3000")
}

// Custom error handler
func customErrorHandler(c *marten.Ctx, err error) {
	// Log the error
	log.Printf("Error: %v (request_id: %s)", err, c.RequestID())

	// Handle different error types
	var validationErr *ValidationError
	var notFoundErr *NotFoundError
	var forbiddenErr *ForbiddenError
	var bindErr *marten.BindError

	switch {
	case errors.As(err, &validationErr):
		c.JSON(400, marten.M{
			"error": "validation_error",
			"details": marten.M{
				"field":   validationErr.Field,
				"message": validationErr.Message,
			},
		})

	case errors.As(err, &notFoundErr):
		c.JSON(404, marten.M{
			"error":    "not_found",
			"resource": notFoundErr.Resource,
			"id":       notFoundErr.ID,
		})

	case errors.As(err, &forbiddenErr):
		c.JSON(403, marten.M{
			"error":   "forbidden",
			"message": forbiddenErr.Message,
		})

	case errors.As(err, &bindErr):
		c.JSON(400, marten.M{
			"error":   "bad_request",
			"message": bindErr.Message,
		})

	default:
		// Generic error
		c.JSON(500, marten.M{
			"error":      "internal_error",
			"message":    "An unexpected error occurred",
			"request_id": c.RequestID(),
		})
	}
}

func getUser(c *marten.Ctx) error {
	id := c.Param("id")

	// Simulate user lookup
	if id == "999" {
		return &NotFoundError{Resource: "user", ID: id}
	}

	return c.OK(marten.M{
		"id":    id,
		"name":  "John Doe",
		"email": "john@example.com",
	})
}

func createUser(c *marten.Ctx) error {
	var input struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Age   int    `json:"age"`
	}

	if err := c.Bind(&input); err != nil {
		return err // Will be handled by customErrorHandler
	}

	// Validation
	if input.Name == "" {
		return &ValidationError{Field: "name", Message: "is required"}
	}
	if input.Email == "" {
		return &ValidationError{Field: "email", Message: "is required"}
	}
	if input.Age < 0 || input.Age > 150 {
		return &ValidationError{Field: "age", Message: "must be between 0 and 150"}
	}

	return c.Created(marten.M{
		"id":    "new-user-id",
		"name":  input.Name,
		"email": input.Email,
		"age":   input.Age,
	})
}

func deleteUser(c *marten.Ctx) error {
	id := c.Param("id")

	// Simulate permission check
	if id == "123" {
		return &ForbiddenError{Message: "cannot delete this user"}
	}

	return c.NoContent()
}

func validateInput(c *marten.Ctx) error {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	// Use BindValid for combined binding and validation
	err := c.BindValid(&input, func() error {
		var errs []ValidationError

		if len(input.Username) < 3 {
			errs = append(errs, ValidationError{
				Field:   "username",
				Message: "must be at least 3 characters",
			})
		}

		if len(input.Password) < 8 {
			errs = append(errs, ValidationError{
				Field:   "password",
				Message: "must be at least 8 characters",
			})
		}

		if input.Email == "" {
			errs = append(errs, ValidationError{
				Field:   "email",
				Message: "is required",
			})
		}

		if len(errs) > 0 {
			// Return first error (or you could return all)
			return &errs[0]
		}

		return nil
	})

	if err != nil {
		return err
	}

	return c.OK(marten.M{
		"message": "validation passed",
		"data":    input,
	})
}
