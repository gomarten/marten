package middleware

import "github.com/gomarten/marten"

// NoCache sets headers to prevent caching.
func NoCache(next marten.Handler) marten.Handler {
	return func(c *marten.Ctx) error {
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Header("Surrogate-Control", "no-store")
		return next(c)
	}
}
