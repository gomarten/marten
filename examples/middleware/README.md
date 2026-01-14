# Middleware Example

Demonstrates built-in middleware: Logger, Recover, CORS, RateLimit, BasicAuth, Timeout, ETag, Compress, Secure, and custom middleware.

## v0.1.2 Features

- `OnStart` / `OnShutdown` lifecycle hooks
- `LoggerWithConfig` with colored output
- `RecoverJSON` for JSON error responses
- `RateLimitConfig.OnLimitReached` custom response
- `TimeoutWithConfig` with custom timeout response
- CORS wildcard subdomain support (`*.example.com`)
- Form binding with `application/x-www-form-urlencoded`

## Run

```bash
go run .
```

## Endpoints

- `GET /` - Basic response with request ID
- `POST /form` - Form binding demo
- `GET /api/limited` - Rate limited (10 req/min)
- `GET /admin/dashboard` - Basic auth protected (admin:secret123)
- `GET /slow/task` - 2s timeout with custom response
- `GET /cached/data` - ETag caching
- `GET /nocache/data` - No-cache headers
- `GET /panic` - Test RecoverJSON middleware
