package middleware

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gomarten/marten"
)

// RecoverConfig configures the recover middleware.
type RecoverConfig struct {
	// OnPanic is called when a panic is recovered.
	// If nil, a default 500 response is sent.
	OnPanic func(c *marten.Ctx, err any) error
	// LogPanics enables logging of panics (default: true)
	LogPanics bool
}

// DefaultRecoverConfig returns sensible defaults.
func DefaultRecoverConfig() RecoverConfig {
	return RecoverConfig{
		LogPanics: true,
	}
}

// Recover catches panics and returns 500.
func Recover(next marten.Handler) marten.Handler {
	return RecoverWithConfig(DefaultRecoverConfig())(next)
}

// RecoverWithConfig returns a recover middleware with custom config.
func RecoverWithConfig(cfg RecoverConfig) marten.Middleware {
	return func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) (err error) {
			defer func() {
				if r := recover(); r != nil {
					if cfg.LogPanics {
						log.Printf("panic recovered: %v", r)
					}

					if cfg.OnPanic != nil {
						err = cfg.OnPanic(c, r)
					} else {
						err = c.Text(http.StatusInternalServerError, "Internal Server Error")
					}
				}
			}()
			return next(c)
		}
	}
}

// RecoverWithHandler creates a recover middleware with a custom panic handler.
func RecoverWithHandler(handler func(c *marten.Ctx, err any) error) marten.Middleware {
	return RecoverWithConfig(RecoverConfig{
		OnPanic:   handler,
		LogPanics: true,
	})
}

// RecoverJSON returns a recover middleware that returns JSON errors.
func RecoverJSON(next marten.Handler) marten.Handler {
	return RecoverWithConfig(RecoverConfig{
		LogPanics: true,
		OnPanic: func(c *marten.Ctx, err any) error {
			return c.JSON(http.StatusInternalServerError, marten.M{
				"error":   "internal server error",
				"message": fmt.Sprintf("%v", err),
			})
		},
	})(next)
}
