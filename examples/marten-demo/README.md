# Gopher Notes

<p align="center">
  <img src="public/gopher.svg" alt="Gopher" width="120">
</p>

A simple note-taking app built with [Marten](https://github.com/gomarten/marten).

## Run

```bash
go mod tidy
go run .
```

Server starts at http://localhost:8080

## Features

- User registration and login
- Create, view, and delete notes
- Clean, responsive UI
- REST API with JSON responses
- Cookie-based web auth + Bearer token API auth

## Web Pages

- `/` - Home page with fortune
- `/login` - Login form
- `/register` - Registration form
- `/dashboard` - Notes dashboard (requires login)
- `/api-docs` - API documentation

## REST API

### Public

```bash
# Health check
curl http://localhost:8080/health

# Get a fortune
curl http://localhost:8080/fortune

# ASCII gopher
curl http://localhost:8080/gopher
```

### Auth

```bash
# Register
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username": "gopher", "password": "secret123"}'

# Login (save the token!)
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "gopher", "password": "secret123"}'
```

### Notes (requires Bearer token)

```bash
TOKEN="your-token-here"

# List notes
curl http://localhost:8080/api/notes \
  -H "Authorization: Bearer $TOKEN"

# Create a note
curl -X POST http://localhost:8080/api/notes \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title": "My Note", "content": "Hello Marten!"}'

# Delete a note
curl -X DELETE http://localhost:8080/api/notes/NOTE_ID \
  -H "Authorization: Bearer $TOKEN"
```

## Marten Features Demonstrated

- Route groups (`/auth`, `/api`)
- Middleware (Logger, Recover, CORS, RateLimit)
- Custom auth middleware (cookie + bearer token)
- HTML templates with `c.HTML()`
- Form handling (`c.FormValue()`)
- JSON binding (`c.Bind()`)
- Response helpers (`c.OK()`, `c.Created()`, `c.BadRequest()`)
- Path parameters (`c.Param("id")`)
- Request-scoped storage (`c.Set()`, `c.GetString()`)
- Cookies (`c.Request.Cookie()`, `http.SetCookie()`)

## Credits

Gopher artwork by [Egon Elbre](https://github.com/egonelbre/gophers) - CC0 Public Domain.
