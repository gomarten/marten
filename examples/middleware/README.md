# Middleware Example

Demonstrates built-in middleware: Logger, Recover, CORS, RateLimit, BasicAuth, Timeout, ETag, Compress, Secure and custom middleware.

## Run

```bash
go run .
```

## Endpoints

- `GET /` - Basic response with request ID
- `GET /api/limited` - Rate limited (10 req/min)
- `GET /admin/dashboard` - Basic auth protected (admin:secret123)
- `GET /slow/task` - 2s timeout
- `GET /cached/data` - ETag caching
- `GET /nocache/data` - No-cache headers
