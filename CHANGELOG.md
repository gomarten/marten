# Changelog

All notable changes to Marten.

## [0.1.3] - 2026-01-18

### Added

- **Static file serving middleware** - `middleware.Static()` for serving static files with features:
  - Automatic content-type detection
  - Directory index serving (index.html)
  - Directory browsing (optional)
  - If-Modified-Since caching support
  - Directory traversal prevention
  - Configurable URL prefix
  - Custom 404 handlers
  - HEAD request support
- 75 comprehensive test cases (325 total) covering edge cases, stress scenarios, and integration workflows

### Fixed

- **Router**: Fixed wildcard routes not matching when accessed with trailing slash (e.g., `/files/` now correctly matches `/files/*filepath` with empty filepath parameter)
- **Router**: Fixed group prefix trailing slash normalization - groups created with trailing slash (e.g., `app.Group("/api/")`) now route correctly
- **Router**: Fixed group path concatenation - paths without leading slash (e.g., `api.GET("users", ...)`) now properly combine with group prefix
- **Context**: Fixed `Stream()` panic when nil reader is provided - now handles gracefully by writing headers only
- **Middleware**: Fixed timeout middleware race condition when handler writes response after timeout - handlers now properly check context cancellation

### Improved

- Better path normalization in group prefixes
- More robust wildcard route matching
- Enhanced test coverage across router, context, middleware, and concurrent operations
- Added stress tests for high concurrency (1,000+ requests) and memory management (10,000+ requests)
- Added integration tests for real-world workflows (CRUD, auth, file uploads)

## [0.1.2] - 2026-01-14

### Added

- `OnStart()` and `OnShutdown()` lifecycle hooks for App
- `LoggerConfig.EnableColors` for colored terminal output
- `LoggerConfig.JSONFormat` for JSON-formatted logs
- `RecoverWithConfig()` with custom panic handler
- `RecoverWithHandler()` convenience function
- `RecoverJSON` middleware for JSON error responses
- `RateLimitConfig.OnLimitReached` for custom rate limit responses
- CORS wildcard subdomain support (e.g., `*.example.com`)
- `Bind()` now supports `application/x-www-form-urlencoded`
- `Bind()` now supports `multipart/form-data`

### Fixed

- Context pool reset now ensures all fields are cleared between requests
- `Bind()` now returns error for empty request body
- `Bind()` now checks Content-Type header before parsing

### Changed

- `DefaultLoggerConfig()` no longer sets a default Format function

## [0.1.1] - 2026-01-09

### Added

- `HEAD()` and `OPTIONS()` route methods on Router and Group
- `Routes()` method to list all registered routes for debugging
- `c.GetHeader()` method to read request headers
- `c.Written()` method to check if response has been written
- `c.HTML()` method for HTML responses
- `c.Blob()` method for binary responses
- `c.Stream()` method for streaming responses from io.Reader
- `c.QueryParams()` method to get all query parameters as url.Values
- `LoggerWithConfig()` for configurable logging (custom output, format, skip)
- `NewRateLimiter()` with `Stop()` method for proper cleanup
- `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` headers
- `Retry-After` header when rate limit exceeded
- `Skip` option for RateLimit and Logger middleware
- `ExposeHeaders` and `MaxAge` options for CORS config
- `SetTrailingSlash()` for configurable trailing slash handling (Ignore, Redirect, Strict)
- Route conflict detection - panics when registering conflicting param routes (e.g., `:id` vs `:name`)
- `SECURITY.md` security policy

### Fixed

- Timeout middleware goroutine leak - now properly handles cancellation
- RateLimit cleanup goroutine now stoppable via `Stop()` method
- Compress middleware now flushes buffered data under MinSize threshold
- Group middleware slice mutation - no longer mutates original slice
- CORS middleware now sets `Vary: Origin` header for proper caching
- ETag middleware now copies original response headers
- BodyLimit middleware now handles chunked encoding (ContentLength = -1)
- Router now returns 405 Method Not Allowed (with `Allow` header) instead of 404 when path exists but method doesn't match

### Changed

- Timeout middleware checks `Written()` before sending timeout response

## [0.1.0] - 2026-01-08

### Added

- Core routing with radix tree
- Route groups with prefix and middleware
- Path parameters (`:id`) and wildcards (`*filepath`)
- Global and route-specific middleware
- Context with JSON/Text responses
- Response helpers: `OK()`, `Created()`, `NoContent()`, `BadRequest()`, `Unauthorized()`, `Forbidden()`, `NotFound()`, `ServerError()`
- Typed parameter helpers: `ParamInt()`, `ParamInt64()`, `QueryInt()`, `QueryInt64()`, `QueryBool()`
- Request helpers: `ClientIP()`, `Bearer()`, `RequestID()`, `IsJSON()`, `IsAJAX()`
- Request-scoped storage: `Set()`, `Get()`, `GetString()`, `GetInt()`, `GetBool()`
- Cookie helpers: `Cookie()`, `SetCookie()`
- Form helpers: `FormValue()`, `File()`
- Convenience types: `marten.M`, `marten.E()`
- `BindValid()` for validation
- Graceful shutdown with `RunGraceful()`
- 13 built-in middleware: Logger, Recover, CORS, RateLimit, BasicAuth, Timeout, Secure, BodyLimit, Compress, ETag, RequestID, NoCache
