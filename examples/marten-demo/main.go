package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/gomarten/marten"
	"github.com/gomarten/marten/middleware"
)

// In-memory storage
var (
	users    = make(map[string]*User)
	notes    = make(map[string][]*Note)
	sessions = make(map[string]*Session)
	mu       sync.RWMutex
)

type User struct {
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Session struct {
	Username  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

const (
	sessionDuration    = 24 * time.Hour
	minUsernameLength  = 3
	maxUsernameLength  = 32
	minPasswordLength  = 8
	maxPasswordLength  = 128
	maxTitleLength     = 200
	maxContentLength   = 10000
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func main() {
	app := marten.New()

	// Global middleware
	app.Use(middleware.Logger)
	app.Use(middleware.Recover)
	app.Use(middleware.Secure(middleware.SecureConfig{
		XSSProtection:      "1; mode=block",
		ContentTypeNosniff: "nosniff",
		XFrameOptions:      "DENY",
	}))
	app.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:8080"},
		AllowMethods:     []string{"GET", "POST", "DELETE"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// Rate limiters
	authRL := middleware.NewRateLimiter(middleware.RateLimitConfig{
		Requests: 5,
		Window:   time.Minute,
	})
	defer authRL.Stop()

	apiRL := middleware.NewRateLimiter(middleware.RateLimitConfig{
		Requests: 60,
		Window:   time.Minute,
	})
	defer apiRL.Stop()

	// ============ WEB PAGES ============
	app.GET("/", homePage)
	app.GET("/login", loginPage)
	app.POST("/login", loginSubmit, authRL.Middleware())
	app.GET("/register", registerPage)
	app.POST("/register", registerSubmit, authRL.Middleware())
	app.GET("/logout", logoutPage)
	app.GET("/dashboard", dashboardPage, webAuth)
	app.POST("/notes", createNotePage, webAuth)
	app.POST("/notes/:id/delete", deleteNotePage, webAuth)
	app.GET("/api-docs", apiDocsPage)

	// Static files
	app.GET("/public/*filepath", serveStatic)

	// ============ JSON API ============
	app.GET("/health", health)
	app.GET("/fortune", fortune)
	app.GET("/gopher", gopher)

	// Auth API (rate limited)
	auth := app.Group("/auth")
	auth.Use(authRL.Middleware())
	auth.POST("/register", registerAPI)
	auth.POST("/login", loginAPI)
	auth.POST("/logout", logoutAPI, apiAuth)

	// Protected API routes
	api := app.Group("/api")
	api.Use(apiAuth)
	api.Use(apiRL.Middleware())
	api.GET("/me", getMe)
	api.GET("/notes", listNotesAPI)
	api.POST("/notes", createNoteAPI)
	api.GET("/notes/:id", getNoteAPI)
	api.DELETE("/notes/:id", deleteNoteAPI)

	log.Println("Gopher Notes running on http://localhost:8080")
	log.Fatal(app.Run(":8080"))
}

// ============ SECURITY HELPERS ============

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func verifyPassword(password, hash string) bool {
	computed := hashPassword(password)
	return subtle.ConstantTimeCompare([]byte(computed), []byte(hash)) == 1
}

func generateSecureToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate secure token")
	}
	return hex.EncodeToString(b)
}

func createSession(username string) string {
	token := generateSecureToken()
	now := time.Now()

	mu.Lock()
	sessions[token] = &Session{
		Username:  username,
		CreatedAt: now,
		ExpiresAt: now.Add(sessionDuration),
	}
	mu.Unlock()

	return token
}

func validateSession(token string) (string, bool) {
	if token == "" {
		return "", false
	}

	mu.RLock()
	session, exists := sessions[token]
	mu.RUnlock()

	if !exists {
		return "", false
	}

	if time.Now().After(session.ExpiresAt) {
		mu.Lock()
		delete(sessions, token)
		mu.Unlock()
		return "", false
	}

	return session.Username, true
}

func invalidateSession(token string) {
	mu.Lock()
	delete(sessions, token)
	mu.Unlock()
}

// ============ VALIDATION HELPERS ============

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func validateUsername(username string) *ValidationError {
	username = strings.TrimSpace(username)

	if username == "" {
		return &ValidationError{"username", "Username is required"}
	}
	if len(username) < minUsernameLength {
		return &ValidationError{"username", "Username must be at least 3 characters"}
	}
	if len(username) > maxUsernameLength {
		return &ValidationError{"username", "Username must be at most 32 characters"}
	}
	if !usernameRegex.MatchString(username) {
		return &ValidationError{"username", "Username can only contain letters, numbers, and underscores"}
	}

	// Check for reserved usernames
	reserved := []string{"admin", "root", "system", "api", "auth", "login", "logout", "register"}
	lower := strings.ToLower(username)
	for _, r := range reserved {
		if lower == r {
			return &ValidationError{"username", "This username is reserved"}
		}
	}

	return nil
}

func validatePassword(password string) *ValidationError {
	if password == "" {
		return &ValidationError{"password", "Password is required"}
	}
	if len(password) < minPasswordLength {
		return &ValidationError{"password", "Password must be at least 8 characters"}
	}
	if len(password) > maxPasswordLength {
		return &ValidationError{"password", "Password is too long"}
	}

	var hasUpper, hasLower, hasDigit bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return &ValidationError{"password", "Password must contain uppercase, lowercase, and a number"}
	}

	return nil
}

func validateNoteTitle(title string) *ValidationError {
	title = strings.TrimSpace(title)
	if title == "" {
		return &ValidationError{"title", "Title is required"}
	}
	if len(title) > maxTitleLength {
		return &ValidationError{"title", "Title is too long (max 200 characters)"}
	}
	return nil
}

func validateNoteContent(content string) *ValidationError {
	if len(content) > maxContentLength {
		return &ValidationError{"content", "Content is too long (max 10000 characters)"}
	}
	return nil
}

func sanitizeInput(s string) string {
	return strings.TrimSpace(s)
}

// ============ TEMPLATE HELPERS ============

func render(c *marten.Ctx, name string, data map[string]any) error {
	t := template.Must(template.ParseFiles(
		"templates/base.html",
		"templates/"+name,
	))
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(200)
	return t.ExecuteTemplate(c.Writer, "base.html", data)
}

// ============ AUTH MIDDLEWARE ============

func webAuth(next marten.Handler) marten.Handler {
	return func(c *marten.Ctx) error {
		cookie, err := c.Request.Cookie("token")
		if err != nil || cookie.Value == "" {
			http.Redirect(c.Writer, c.Request, "/login", http.StatusSeeOther)
			return nil
		}

		username, valid := validateSession(cookie.Value)
		if !valid {
			// Clear invalid cookie
			http.SetCookie(c.Writer, &http.Cookie{
				Name:     "token",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
			})
			http.Redirect(c.Writer, c.Request, "/login", http.StatusSeeOther)
			return nil
		}

		c.Set("username", username)
		c.Set("token", cookie.Value)
		return next(c)
	}
}

func apiAuth(next marten.Handler) marten.Handler {
	return func(c *marten.Ctx) error {
		token := c.Bearer()
		if token == "" {
			return c.JSON(401, marten.M{
				"error": "authentication required",
				"code":  "MISSING_TOKEN",
			})
		}

		username, valid := validateSession(token)
		if !valid {
			return c.JSON(401, marten.M{
				"error": "invalid or expired token",
				"code":  "INVALID_TOKEN",
			})
		}

		c.Set("username", username)
		return next(c)
	}
}

// ============ WEB PAGE HANDLERS ============

func homePage(c *marten.Ctx) error {
	// If already logged in, redirect to dashboard
	if cookie, err := c.Request.Cookie("token"); err == nil {
		if _, valid := validateSession(cookie.Value); valid {
			http.Redirect(c.Writer, c.Request, "/dashboard", http.StatusSeeOther)
			return nil
		}
	}

	return render(c, "home.html", map[string]any{
		"Title":   "Home",
		"Fortune": getRandomFortune(),
	})
}

func loginPage(c *marten.Ctx) error {
	// If already logged in, redirect to dashboard
	if cookie, err := c.Request.Cookie("token"); err == nil {
		if _, valid := validateSession(cookie.Value); valid {
			http.Redirect(c.Writer, c.Request, "/dashboard", http.StatusSeeOther)
			return nil
		}
	}

	return render(c, "login.html", map[string]any{
		"Title": "Login",
	})
}

func loginSubmit(c *marten.Ctx) error {
	username := sanitizeInput(c.FormValue("username"))
	password := c.FormValue("password")

	// Generic error message to prevent username enumeration
	genericError := "Invalid username or password"

	if username == "" || password == "" {
		return render(c, "login.html", map[string]any{
			"Title": "Login",
			"Error": genericError,
		})
	}

	mu.RLock()
	user, exists := users[strings.ToLower(username)]
	mu.RUnlock()

	// Constant-time comparison to prevent timing attacks
	if !exists || !verifyPassword(password, user.PasswordHash) {
		// Add small delay to prevent brute force
		time.Sleep(100 * time.Millisecond)
		return render(c, "login.html", map[string]any{
			"Title": "Login",
			"Error": genericError,
		})
	}

	token := createSession(user.Username)

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(sessionDuration.Seconds()),
	})

	http.Redirect(c.Writer, c.Request, "/dashboard", http.StatusSeeOther)
	return nil
}

func registerPage(c *marten.Ctx) error {
	// If already logged in, redirect to dashboard
	if cookie, err := c.Request.Cookie("token"); err == nil {
		if _, valid := validateSession(cookie.Value); valid {
			http.Redirect(c.Writer, c.Request, "/dashboard", http.StatusSeeOther)
			return nil
		}
	}

	return render(c, "register.html", map[string]any{
		"Title": "Register",
	})
}

func registerSubmit(c *marten.Ctx) error {
	username := sanitizeInput(c.FormValue("username"))
	password := c.FormValue("password")

	// Validate username
	if err := validateUsername(username); err != nil {
		return render(c, "register.html", map[string]any{
			"Title": "Register",
			"Error": err.Message,
		})
	}

	// Validate password
	if err := validatePassword(password); err != nil {
		return render(c, "register.html", map[string]any{
			"Title": "Register",
			"Error": err.Message,
		})
	}

	normalizedUsername := strings.ToLower(username)

	mu.Lock()
	if _, exists := users[normalizedUsername]; exists {
		mu.Unlock()
		return render(c, "register.html", map[string]any{
			"Title": "Register",
			"Error": "Username already taken",
		})
	}

	users[normalizedUsername] = &User{
		Username:     username, // Preserve original case for display
		PasswordHash: hashPassword(password),
		CreatedAt:    time.Now(),
	}
	notes[normalizedUsername] = []*Note{}
	mu.Unlock()

	http.Redirect(c.Writer, c.Request, "/login?registered=1", http.StatusSeeOther)
	return nil
}

func logoutPage(c *marten.Ctx) error {
	if cookie, err := c.Request.Cookie("token"); err == nil && cookie.Value != "" {
		invalidateSession(cookie.Value)
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	http.Redirect(c.Writer, c.Request, "/", http.StatusSeeOther)
	return nil
}

func dashboardPage(c *marten.Ctx) error {
	username := c.GetString("username")

	mu.RLock()
	user := users[strings.ToLower(username)]
	userNotes := notes[strings.ToLower(username)]
	mu.RUnlock()

	displayName := username
	if user != nil {
		displayName = user.Username
	}

	return render(c, "dashboard.html", map[string]any{
		"Title":    "Dashboard",
		"Username": displayName,
		"Notes":    userNotes,
		"Success":  c.Query("success"),
	})
}

func createNotePage(c *marten.Ctx) error {
	username := strings.ToLower(c.GetString("username"))
	title := sanitizeInput(c.FormValue("title"))
	content := sanitizeInput(c.FormValue("content"))

	if err := validateNoteTitle(title); err != nil {
		http.Redirect(c.Writer, c.Request, "/dashboard?error="+err.Message, http.StatusSeeOther)
		return nil
	}

	if err := validateNoteContent(content); err != nil {
		http.Redirect(c.Writer, c.Request, "/dashboard?error="+err.Message, http.StatusSeeOther)
		return nil
	}

	note := &Note{
		ID:        generateSecureToken()[:16],
		Title:     title,
		Content:   content,
		CreatedAt: time.Now(),
	}

	mu.Lock()
	notes[username] = append(notes[username], note)
	mu.Unlock()

	http.Redirect(c.Writer, c.Request, "/dashboard?success=Note+created", http.StatusSeeOther)
	return nil
}

func deleteNotePage(c *marten.Ctx) error {
	username := strings.ToLower(c.GetString("username"))
	noteID := c.Param("id")

	if noteID == "" {
		http.Redirect(c.Writer, c.Request, "/dashboard", http.StatusSeeOther)
		return nil
	}

	mu.Lock()
	userNotes := notes[username]
	for i, n := range userNotes {
		if n.ID == noteID {
			notes[username] = append(userNotes[:i], userNotes[i+1:]...)
			break
		}
	}
	mu.Unlock()

	http.Redirect(c.Writer, c.Request, "/dashboard?success=Note+deleted", http.StatusSeeOther)
	return nil
}

func apiDocsPage(c *marten.Ctx) error {
	return render(c, "api-docs.html", map[string]any{
		"Title": "API Docs",
	})
}

// ============ JSON API HANDLERS ============

func health(c *marten.Ctx) error {
	return c.OK(marten.M{"status": "ok", "time": time.Now()})
}

func registerAPI(c *marten.Ctx) error {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.Bind(&input); err != nil {
		return c.JSON(400, marten.M{
			"error": "invalid request body",
			"code":  "INVALID_JSON",
		})
	}

	input.Username = sanitizeInput(input.Username)

	// Validate username
	if err := validateUsername(input.Username); err != nil {
		return c.JSON(400, marten.M{
			"error": err.Message,
			"field": err.Field,
			"code":  "VALIDATION_ERROR",
		})
	}

	// Validate password
	if err := validatePassword(input.Password); err != nil {
		return c.JSON(400, marten.M{
			"error": err.Message,
			"field": err.Field,
			"code":  "VALIDATION_ERROR",
		})
	}

	normalizedUsername := strings.ToLower(input.Username)

	mu.Lock()
	if _, exists := users[normalizedUsername]; exists {
		mu.Unlock()
		return c.JSON(409, marten.M{
			"error": "username already taken",
			"code":  "USERNAME_EXISTS",
		})
	}

	users[normalizedUsername] = &User{
		Username:     input.Username,
		PasswordHash: hashPassword(input.Password),
		CreatedAt:    time.Now(),
	}
	notes[normalizedUsername] = []*Note{}
	mu.Unlock()

	return c.JSON(201, marten.M{
		"message":  "registration successful",
		"username": input.Username,
	})
}

func loginAPI(c *marten.Ctx) error {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.Bind(&input); err != nil {
		return c.JSON(400, marten.M{
			"error": "invalid request body",
			"code":  "INVALID_JSON",
		})
	}

	input.Username = sanitizeInput(input.Username)

	if input.Username == "" || input.Password == "" {
		return c.JSON(400, marten.M{
			"error": "username and password are required",
			"code":  "MISSING_CREDENTIALS",
		})
	}

	mu.RLock()
	user, exists := users[strings.ToLower(input.Username)]
	mu.RUnlock()

	if !exists || !verifyPassword(input.Password, user.PasswordHash) {
		time.Sleep(100 * time.Millisecond) // Prevent timing attacks
		return c.JSON(401, marten.M{
			"error": "invalid credentials",
			"code":  "INVALID_CREDENTIALS",
		})
	}

	token := createSession(user.Username)

	return c.OK(marten.M{
		"message":    "login successful",
		"token":      token,
		"expires_in": int(sessionDuration.Seconds()),
	})
}

func logoutAPI(c *marten.Ctx) error {
	token := c.Bearer()
	invalidateSession(token)
	return c.OK(marten.M{"message": "logged out"})
}

func getMe(c *marten.Ctx) error {
	username := c.GetString("username")
	normalizedUsername := strings.ToLower(username)

	mu.RLock()
	user := users[normalizedUsername]
	noteCount := len(notes[normalizedUsername])
	mu.RUnlock()

	if user == nil {
		return c.JSON(404, marten.M{"error": "user not found"})
	}

	return c.OK(marten.M{
		"username":    user.Username,
		"notes_count": noteCount,
		"created_at":  user.CreatedAt,
	})
}

func listNotesAPI(c *marten.Ctx) error {
	username := strings.ToLower(c.GetString("username"))

	mu.RLock()
	userNotes := notes[username]
	mu.RUnlock()

	if userNotes == nil {
		userNotes = []*Note{}
	}

	return c.OK(marten.M{
		"notes": userNotes,
		"count": len(userNotes),
	})
}

func createNoteAPI(c *marten.Ctx) error {
	username := strings.ToLower(c.GetString("username"))

	var input struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}

	if err := c.Bind(&input); err != nil {
		return c.JSON(400, marten.M{
			"error": "invalid request body",
			"code":  "INVALID_JSON",
		})
	}

	input.Title = sanitizeInput(input.Title)
	input.Content = sanitizeInput(input.Content)

	if err := validateNoteTitle(input.Title); err != nil {
		return c.JSON(400, marten.M{
			"error": err.Message,
			"field": err.Field,
			"code":  "VALIDATION_ERROR",
		})
	}

	if err := validateNoteContent(input.Content); err != nil {
		return c.JSON(400, marten.M{
			"error": err.Message,
			"field": err.Field,
			"code":  "VALIDATION_ERROR",
		})
	}

	note := &Note{
		ID:        generateSecureToken()[:16],
		Title:     input.Title,
		Content:   input.Content,
		CreatedAt: time.Now(),
	}

	mu.Lock()
	notes[username] = append(notes[username], note)
	mu.Unlock()

	return c.JSON(201, note)
}

func getNoteAPI(c *marten.Ctx) error {
	username := strings.ToLower(c.GetString("username"))
	noteID := c.Param("id")

	if noteID == "" {
		return c.JSON(400, marten.M{"error": "note ID is required"})
	}

	mu.RLock()
	userNotes := notes[username]
	mu.RUnlock()

	for _, n := range userNotes {
		if n.ID == noteID {
			return c.OK(n)
		}
	}

	return c.JSON(404, marten.M{
		"error": "note not found",
		"code":  "NOT_FOUND",
	})
}

func deleteNoteAPI(c *marten.Ctx) error {
	username := strings.ToLower(c.GetString("username"))
	noteID := c.Param("id")

	if noteID == "" {
		return c.JSON(400, marten.M{"error": "note ID is required"})
	}

	mu.Lock()
	defer mu.Unlock()

	userNotes := notes[username]
	for i, n := range userNotes {
		if n.ID == noteID {
			notes[username] = append(userNotes[:i], userNotes[i+1:]...)
			return c.OK(marten.M{"message": "note deleted"})
		}
	}

	return c.JSON(404, marten.M{
		"error": "note not found",
		"code":  "NOT_FOUND",
	})
}

// ============ FUN ENDPOINTS ============

var fortunes = []string{
	"A gopher a day keeps the bugs away.",
	"Concurrency is not parallelism, but it's still awesome.",
	"Keep calm and go fmt.",
	"In Go we trust.",
	"Simplicity is the ultimate sophistication.",
	"Error handling is a feature, not a bug.",
	"Channels: because sharing memory is overrated.",
	"go run . && be happy",
}

func getRandomFortune() string {
	b := make([]byte, 1)
	_, _ = rand.Read(b)
	return fortunes[int(b[0])%len(fortunes)]
}

func fortune(c *marten.Ctx) error {
	return c.OK(marten.M{"fortune": getRandomFortune()})
}

func gopher(c *marten.Ctx) error {
	http.ServeFile(c.Writer, c.Request, "public/gopher.svg")
	return nil
}

func serveStatic(c *marten.Ctx) error {
	filepath := c.Param("filepath")
	// Prevent directory traversal
	if strings.Contains(filepath, "..") {
		return c.JSON(400, marten.M{"error": "invalid path"})
	}
	http.ServeFile(c.Writer, c.Request, "public/"+filepath)
	return nil
}
