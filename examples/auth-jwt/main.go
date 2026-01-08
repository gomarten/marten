// JWT Authentication example
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/gomarten/marten"
	"github.com/gomarten/marten/middleware"
)

// Simple JWT implementation (use a proper library in production)
var jwtSecret = []byte("your-secret-key-change-in-production")

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Exp    int64  `json:"exp"`
}

func main() {
	app := marten.New()

	app.Use(middleware.Logger, middleware.Recover)
	app.Use(middleware.CORS(middleware.DefaultCORSConfig()))

	// Public routes
	app.POST("/auth/login", login)
	app.POST("/auth/register", register)

	// Protected routes
	protected := app.Group("/api")
	protected.Use(JWTMiddleware)
	{
		protected.GET("/me", getMe)
		protected.GET("/profile", getProfile)
		protected.PUT("/profile", updateProfile)
	}

	// Admin routes (JWT + role check)
	admin := app.Group("/admin")
	admin.Use(JWTMiddleware, AdminMiddleware)
	{
		admin.GET("/users", listAllUsers)
		admin.DELETE("/users/:id", deleteUser)
	}

	log.Println("Auth example running on http://localhost:3000")
	app.Run(":3000")
}

// --- Auth Handlers ---

func login(c *marten.Ctx) error {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.Bind(&input); err != nil {
		return c.BadRequest("invalid request body")
	}

	// In production, verify against database
	if input.Email != "user@example.com" || input.Password != "password123" {
		return c.Unauthorized("invalid credentials")
	}

	// Generate JWT
	token, err := generateJWT("user-123", input.Email)
	if err != nil {
		return c.ServerError("failed to generate token")
	}

	return c.OK(marten.M{
		"token":      token,
		"expires_in": 3600,
	})
}

func register(c *marten.Ctx) error {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}

	if err := c.BindValid(&input, func() error {
		if input.Email == "" {
			return &marten.BindError{Message: "email is required"}
		}
		if input.Password == "" || len(input.Password) < 8 {
			return &marten.BindError{Message: "password must be at least 8 characters"}
		}
		return nil
	}); err != nil {
		return c.BadRequest(err.Error())
	}

	// In production, save to database
	token, _ := generateJWT("new-user-id", input.Email)

	return c.Created(marten.M{
		"message": "user created",
		"token":   token,
	})
}

func getMe(c *marten.Ctx) error {
	userID := c.GetString("user_id")
	email := c.GetString("email")

	return c.OK(marten.M{
		"user_id": userID,
		"email":   email,
	})
}

func getProfile(c *marten.Ctx) error {
	userID := c.GetString("user_id")

	// In production, fetch from database
	return c.OK(marten.M{
		"user_id": userID,
		"name":    "John Doe",
		"email":   c.GetString("email"),
		"bio":     "Software developer",
	})
}

func updateProfile(c *marten.Ctx) error {
	var input struct {
		Name string `json:"name"`
		Bio  string `json:"bio"`
	}

	if err := c.Bind(&input); err != nil {
		return c.BadRequest("invalid request body")
	}

	// In production, update database
	return c.OK(marten.M{
		"message": "profile updated",
		"name":    input.Name,
		"bio":     input.Bio,
	})
}

func listAllUsers(c *marten.Ctx) error {
	// Admin only endpoint
	return c.OK(marten.M{
		"users": []marten.M{
			{"id": "1", "email": "user1@example.com"},
			{"id": "2", "email": "user2@example.com"},
		},
	})
}

func deleteUser(c *marten.Ctx) error {
	id := c.Param("id")
	return c.OK(marten.M{"message": "user " + id + " deleted"})
}

// --- Middleware ---

func JWTMiddleware(next marten.Handler) marten.Handler {
	return func(c *marten.Ctx) error {
		token := c.Bearer()
		if token == "" {
			return c.Unauthorized("missing token")
		}

		claims, err := validateJWT(token)
		if err != nil {
			return c.Unauthorized("invalid token")
		}

		// Store claims in context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)

		return next(c)
	}
}

func AdminMiddleware(next marten.Handler) marten.Handler {
	return func(c *marten.Ctx) error {
		// In production, check user role from database
		email := c.GetString("email")
		if email != "admin@example.com" {
			return c.Forbidden("admin access required")
		}
		return next(c)
	}
}

// --- JWT Helpers (simplified - use a proper library in production) ---

func generateJWT(userID, email string) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	claims := Claims{
		UserID: userID,
		Email:  email,
		Exp:    time.Now().Add(time.Hour).Unix(),
	}
	claimsJSON, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signature := sign(header + "." + payload)

	return header + "." + payload + "." + signature, nil
}

func validateJWT(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, &marten.BindError{Message: "invalid token format"}
	}

	// Verify signature
	expectedSig := sign(parts[0] + "." + parts[1])
	if parts[2] != expectedSig {
		return nil, &marten.BindError{Message: "invalid signature"}
	}

	// Decode claims
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	var claims Claims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, err
	}

	// Check expiration
	if claims.Exp < time.Now().Unix() {
		return nil, &marten.BindError{Message: "token expired"}
	}

	return &claims, nil
}

func sign(data string) string {
	h := hmac.New(sha256.New, jwtSecret)
	h.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
