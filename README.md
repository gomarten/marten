<p align="center">
  <img src="assets/logo.png" alt="Marten" width="400">
</p>

<p align="center">
  <strong>A minimal, zero-dependency web framework for Go.</strong>
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/gomarten/marten"><img src="https://pkg.go.dev/badge/github.com/gomarten/marten.svg" alt="Go Reference"></a>
  <a href="https://goreportcard.com/report/github.com/gomarten/marten"><img src="https://goreportcard.com/badge/github.com/gomarten/marten" alt="Go Report Card"></a>
  <a href="https://github.com/gomarten/marten/actions/workflows/ci.yml"><img src="https://github.com/gomarten/marten/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
</p>

---

Marten is a lightweight HTTP framework built entirely on Go's standard library. No external dependencies. No magic. Just clean, predictable code that gets out of your way.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Features](#features)
- [Routing](#routing)
- [Middleware](#middleware)
- [Context API](#context-api)
- [Configuration](#configuration)
- [Benchmarks](#benchmarks)
- [Examples](#examples)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [License](#license)

## Installation

```bash
go get github.com/gomarten/marten
```

## Quick Start

```go
package main

import (
    "github.com/gomarten/marten"
    "github.com/gomarten/marten/middleware"
)

func main() {
    app := marten.New()
    
    app.Use(middleware.Logger)
    app.Use(middleware.Recover)

    app.GET("/", func(c *marten.Ctx) error {
        return c.OK(marten.M{"message": "Hello, World!"})
    })

    app.GET("/users/:id", func(c *marten.Ctx) error {
        id := c.ParamInt("id")
        return c.OK(marten.M{"id": id})
    })

    app.Run(":8080")
}
```

## Features

| Feature | Description |
|---------|-------------|
| Zero Dependencies | Built entirely on Go's standard library |
| Fast Routing | Radix tree router with path parameters and wildcards |
| Middleware | Chainable middleware with 14 built-in options |
| Context Pooling | Efficient memory reuse for high throughput |
| Response Helpers | `OK()`, `Created()`, `BadRequest()`, `NotFound()`, and more |
| Typed Parameters | `ParamInt()`, `QueryInt()`, `QueryBool()` |
| Graceful Shutdown | Built-in support via `RunGraceful()` |

## Routing

```go
// Path parameters
app.GET("/users/:id", handler)
app.GET("/files/*filepath", handler)

// Route groups
api := app.Group("/api/v1")
api.GET("/users", listUsers)
api.POST("/users", createUser)

// All HTTP methods
app.GET("/resource", handler)
app.POST("/resource", handler)
app.PUT("/resource", handler)
app.DELETE("/resource", handler)
app.PATCH("/resource", handler)
app.HEAD("/resource", handler)
app.OPTIONS("/resource", handler)
```

## Middleware

Built-in middleware:

```go
import "github.com/gomarten/marten/middleware"

app.Use(middleware.Logger)           // Request logging
app.Use(middleware.Recover)          // Panic recovery
app.Use(middleware.CORS(config))     // Cross-origin requests
app.Use(middleware.RateLimit(cfg))   // Rate limiting
app.Use(middleware.BasicAuth(cfg))   // Basic authentication
app.Use(middleware.Timeout(5*time.Second))
app.Use(middleware.Compress(cfg))    // Gzip compression
app.Use(middleware.Secure(cfg))      // Security headers
app.Use(middleware.RequestID)        // Request ID injection
app.Use(middleware.BodyLimit(1*middleware.MB))
app.Use(middleware.ETag)             // ETag caching
app.Use(middleware.NoCache)          // Cache prevention
app.Use(middleware.Static("./public")) // Static file serving
```

Route-specific middleware:

```go
app.GET("/admin", adminHandler, authMiddleware, logMiddleware)
```

## Context API

```go
func handler(c *marten.Ctx) error {
    // Path and query parameters
    id := c.Param("id")
    page := c.QueryInt("page")
    
    // Request data
    ip := c.ClientIP()
    token := c.Bearer()
    
    // JSON binding
    var user User
    if err := c.Bind(&user); err != nil {
        return c.BadRequest("invalid JSON")
    }
    
    // Responses
    return c.OK(data)              // 200
    return c.Created(data)         // 201
    return c.NoContent()           // 204
    return c.BadRequest("error")   // 400
    return c.NotFound("not found") // 404
}
```

## Configuration

```go
// Trailing slash handling
app.SetTrailingSlash(marten.TrailingSlashRedirect)

// Custom 404 handler
app.NotFound(func(c *marten.Ctx) error {
    return c.JSON(404, marten.E("page not found"))
})

// Custom error handler
app.OnError(func(c *marten.Ctx, err error) {
    c.JSON(500, marten.E(err.Error()))
})

// Graceful shutdown
app.RunGraceful(":8080", 10*time.Second)
```

## Benchmarks

Marten performs competitively with Gin and Echo while maintaining zero dependencies.

| Benchmark | Marten | Gin | Echo | Chi |
|-----------|--------|-----|------|-----|
| Static Route | 1,445 ns/op | 1,323 ns/op | 1,421 ns/op | 2,208 ns/op |
| Param Route | 1,536 ns/op | 1,419 ns/op | 1,474 ns/op | 2,520 ns/op |
| JSON Response | 1,651 ns/op | 1,583 ns/op | 1,754 ns/op | 1,890 ns/op |
| JSON Binding | 8,339 ns/op | 8,634 ns/op | 8,766 ns/op | 6,810 ns/op |
| Multi-Param | 1,841 ns/op | 1,511 ns/op | 1,634 ns/op | 2,836 ns/op |

*Tested on Intel Xeon Platinum 8259CL @ 2.50GHz, Go 1.24, Linux*

See [benchmarks/](benchmarks/) for full comparison with Gin, Echo, Chi and Fiber.

## Examples

| Example | Description |
|---------|-------------|
| [basic](examples/basic/) | Hello World, JSON, path and query params |
| [crud-api](examples/crud-api/) | RESTful API with validation |
| [auth-jwt](examples/auth-jwt/) | JWT authentication |
| [middleware](examples/middleware/) | Built-in and custom middleware |
| [groups](examples/groups/) | Route groups and versioning |
| [error-handling](examples/error-handling/) | Custom error types and handlers |
| [file-server](examples/file-server/) | Static files and SPA fallback |
| [marten-demo](examples/marten-demo/) | Full web app with auth and templates |

## Documentation

Full documentation available at [gomarten.github.io/docs](https://gomarten.github.io/docs)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Community & Discussions
Questions or ideas?  
Join the discussion: https://github.com/gomarten/marten/discussions

## License

MIT License. See [LICENSE](LICENSE) for details.
