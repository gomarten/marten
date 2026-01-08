package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gomarten/marten"
)

// RateLimitConfig configures the rate limiter.
type RateLimitConfig struct {
	Requests int
	Window   time.Duration
	KeyFunc  func(*marten.Ctx) string
}

// DefaultRateLimitConfig returns sensible defaults.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Requests: 100,
		Window:   time.Minute,
		KeyFunc:  func(c *marten.Ctx) string { return c.ClientIP() },
	}
}

// RateLimit returns a rate limiting middleware.
func RateLimit(cfg RateLimitConfig) marten.Middleware {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = func(c *marten.Ctx) string { return c.ClientIP() }
	}

	var mu sync.Mutex
	clients := make(map[string]*bucket)

	go func() {
		for {
			time.Sleep(cfg.Window)
			mu.Lock()
			now := time.Now()
			for k, b := range clients {
				if now.Sub(b.reset) > cfg.Window*2 {
					delete(clients, k)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			key := cfg.KeyFunc(c)

			mu.Lock()
			b, ok := clients[key]
			if !ok {
				b = &bucket{reset: time.Now().Add(cfg.Window), remaining: cfg.Requests}
				clients[key] = b
			}

			now := time.Now()
			if now.After(b.reset) {
				b.reset = now.Add(cfg.Window)
				b.remaining = cfg.Requests
			}

			if b.remaining <= 0 {
				mu.Unlock()
				return c.JSON(http.StatusTooManyRequests, marten.E("rate limit exceeded"))
			}

			b.remaining--
			mu.Unlock()

			return next(c)
		}
	}
}

type bucket struct {
	reset     time.Time
	remaining int
}
