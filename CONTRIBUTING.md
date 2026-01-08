# Contributing to Marten

Thanks for your interest in contributing!

## Getting Started

```bash
git clone https://github.com/gomarten/marten.git
cd marten
go test ./tests/...
```

## Making Changes

1. Fork the repo
2. Create a branch (`git checkout -b feature/my-change`)
3. Make your changes
4. Run tests (`go test ./tests/...`)
5. Commit (`git commit -m "Add feature"`)
6. Push (`git push origin feature/my-change`)
7. Open a Pull Request

## Guidelines

- Keep it simple - Marten values minimalism
- No external dependencies - stdlib only
- Write tests for new features
- Run `go fmt` before committing
- Keep commits focused and atomic

## What We're Looking For

- Bug fixes
- Performance improvements
- Documentation improvements
- New middleware (if it fits the philosophy)
- Test coverage

## What We're Not Looking For

- External dependencies
- Breaking API changes
- Features that duplicate stdlib

## Code Style

```go
// Good - clear and minimal
func (c *Ctx) OK(v any) error {
    return c.JSON(http.StatusOK, v)
}

// Avoid - overly complex
func (c *Ctx) OK(v any, opts ...Option) error {
    // ...
}
```

## Questions?

Open an issue for discussion before starting major work.
