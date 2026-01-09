<p align="center">
  <img src="assets/logo.png" alt="Marten" width="400">
</p>

<p align="center">
  <strong>The Go web framework you reach for when you want nothing in the way.</strong>
</p>


<p align="center">
  <a href="https://pkg.go.dev/github.com/gomarten/marten"><img src="https://pkg.go.dev/badge/github.com/gomarten/marten.svg" alt="Go Reference"></a>
  <a href="https://goreportcard.com/report/github.com/gomarten/marten"><img src="https://goreportcard.com/badge/github.com/gomarten/marten" alt="Go Report Card"></a>
  <a href="https://github.com/gomarten/marten/actions/workflows/ci.yml"><img src="https://github.com/gomarten/marten/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/github/go-mod/go-version/gomarten/marten" alt="Go Version"></a>
</p>

---

## Features

- **Zero dependencies** - Only Go's standard library
- **Fast** - Radix tree routing with context pooling
- **Simple** - Clean, chainable API
- **Batteries included** - 13 built-in middleware

## Quick Start

```bash
go get github.com/gomarten/marten
```

```go
package main

import (
    "github.com/gomarten/marten"
    "github.com/gomarten/marten/middleware"
)

func main() {
    app := marten.New()
    app.Use(middleware.Logger, middleware.Recover)

    app.GET("/", func(c *marten.Ctx) error {
        return c.OK(marten.M{"message": "Hello, Marten!"})
    })

    app.GET("/users/:id", func(c *marten.Ctx) error {
        return c.OK(marten.M{"id": c.Param("id")})
    })

    app.Run(":8080")
}
```

## Documentation

Full documentation at **[gomarten.github.io/docs](https://gomarten.github.io/docs)**

## License

MIT
