# Marten

The Go web framework you reach for when you want nothing in the way.

[![Go Reference](https://pkg.go.dev/badge/github.com/gomarten/marten.svg)](https://pkg.go.dev/github.com/gomarten/marten)
[![Go Report Card](https://goreportcard.com/badge/github.com/gomarten/marten)](https://goreportcard.com/report/github.com/gomarten/marten)

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
