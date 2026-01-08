package marten

// Handler is a function that handles an HTTP request.
type Handler func(*Ctx) error

// Middleware wraps a handler with additional behavior.
type Middleware func(Handler) Handler

// Chain composes multiple middleware into a single middleware.
func Chain(mw ...Middleware) Middleware {
	return func(final Handler) Handler {
		for i := len(mw) - 1; i >= 0; i-- {
			final = mw[i](final)
		}
		return final
	}
}
