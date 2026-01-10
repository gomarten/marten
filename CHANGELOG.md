# Changelog

All notable changes to Marten.

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
