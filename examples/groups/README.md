# Route Groups Example

Organizing routes with groups, nested groups and group-specific middleware.

## Run

```bash
go run .
```

## Structure

- `/api/v1/*` - API version 1
- `/api/v2/*` - API version 2 (with version header)
- `/admin/*` - Basic auth protected (admin:secret)
- `/webhooks/*` - Webhook secret required

## Features

- Nested route groups
- Per-group middleware
- API versioning pattern
- Uses `app.Routes()` to list all registered routes
