# Error Handling Example

Custom error types, error handler, and validation patterns.

## Run

```bash
go run .
```

## Endpoints

- `GET /users/:id` - Returns 404 for id=999
- `POST /users` - Validation errors
- `DELETE /users/:id` - Returns 403 for id=123
- `GET /panic` - Panic recovery demo
- `POST /validate` - Input validation

## Features

- Custom error types (ValidationError, NotFoundError, ForbiddenError)
- Global error handler with `app.OnError()`
- Panic recovery middleware
- `BindValid()` for combined binding and validation
