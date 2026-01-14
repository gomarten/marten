package middleware

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gomarten/marten"
)

// RateLimitConfig configures the rate limiter.
type RateLimitConfig struct {
	Requests    int                           // Max requests per window
	Window      time.Duration                 // Time window
	KeyFunc     func(*marten.Ctx) string      // Extract key (default: ClientIP)
	Skip        func(*marten.Ctx) bool        // Skip rate limiting for certain requests
	OnLimitReached func(*marten.Ctx) error    // Custom response when limit is reached
}

// DefaultRateLimitConfig returns sensible defaults.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Requests: 100,
		Window:   time.Minute,
		KeyFunc:  func(c *marten.Ctx) string { return c.ClientIP() },
	}
}

// RateLimiter is a rate limiter that can be stopped.
type RateLimiter struct {
	cfg     RateLimitConfig
	mu      sync.Mutex
	clients map[string]*bucket
	cancel  context.CancelFunc
}

// NewRateLimiter creates a new rate limiter with cleanup goroutine.
func NewRateLimiter(cfg RateLimitConfig) *RateLimiter {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = func(c *marten.Ctx) string { return c.ClientIP() }
	}
	if cfg.Window == 0 {
		cfg.Window = time.Minute
	}
	if cfg.Requests == 0 {
		cfg.Requests = 100
	}

	ctx, cancel := context.WithCancel(context.Background())
	rl := &RateLimiter{
		cfg:     cfg,
		clients: make(map[string]*bucket),
		cancel:  cancel,
	}

	// Cleanup goroutine with cancellation
	go func() {
		ticker := time.NewTicker(cfg.Window)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				rl.cleanup()
			}
		}
	}()

	return rl
}

// Stop stops the rate limiter cleanup goroutine.
func (rl *RateLimiter) Stop() {
	rl.cancel()
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	for k, b := range rl.clients {
		if now.Sub(b.reset) > rl.cfg.Window*2 {
			delete(rl.clients, k)
		}
	}
}

// Middleware returns the rate limiting middleware.
func (rl *RateLimiter) Middleware() marten.Middleware {
	return func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			if rl.cfg.Skip != nil && rl.cfg.Skip(c) {
				return next(c)
			}

			key := rl.cfg.KeyFunc(c)

			rl.mu.Lock()
			b, ok := rl.clients[key]
			if !ok {
				b = &bucket{reset: time.Now().Add(rl.cfg.Window), remaining: rl.cfg.Requests}
				rl.clients[key] = b
			}

			now := time.Now()
			if now.After(b.reset) {
				b.reset = now.Add(rl.cfg.Window)
				b.remaining = rl.cfg.Requests
			}

			// Set rate limit headers
			c.Header("X-RateLimit-Limit", strconv.Itoa(rl.cfg.Requests))
			c.Header("X-RateLimit-Remaining", strconv.Itoa(b.remaining))
			c.Header("X-RateLimit-Reset", strconv.FormatInt(b.reset.Unix(), 10))

			if b.remaining <= 0 {
				rl.mu.Unlock()
				c.Header("Retry-After", strconv.FormatInt(int64(time.Until(b.reset).Seconds()), 10))
				if rl.cfg.OnLimitReached != nil {
					return rl.cfg.OnLimitReached(c)
				}
				return c.JSON(http.StatusTooManyRequests, marten.E("rate limit exceeded"))
			}

			b.remaining--
			rl.mu.Unlock()

			return next(c)
		}
	}
}

// RateLimit returns a rate limiting middleware (convenience function).
// Note: This creates a rate limiter that cannot be stopped. For production,
// use NewRateLimiter() and call Stop() when done.
func RateLimit(cfg RateLimitConfig) marten.Middleware {
	return NewRateLimiter(cfg).Middleware()
}

type bucket struct {
	reset     time.Time
	remaining int
}
