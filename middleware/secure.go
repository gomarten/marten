package middleware

import (
	"fmt"

	"github.com/gomarten/marten"
)

// SecureConfig configures security headers.
type SecureConfig struct {
	XSSProtection         string
	ContentTypeNosniff    string
	XFrameOptions         string
	HSTSMaxAge            int
	HSTSIncludeSubdomains bool
	ContentSecurityPolicy string
	ReferrerPolicy        string
}

// DefaultSecureConfig returns sensible security defaults.
func DefaultSecureConfig() SecureConfig {
	return SecureConfig{
		XSSProtection:      "1; mode=block",
		ContentTypeNosniff: "nosniff",
		XFrameOptions:      "SAMEORIGIN",
		ReferrerPolicy:     "strict-origin-when-cross-origin",
	}
}

// Secure returns a middleware that sets security headers.
func Secure(cfg SecureConfig) marten.Middleware {
	return func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			if cfg.XSSProtection != "" {
				c.Header("X-XSS-Protection", cfg.XSSProtection)
			}
			if cfg.ContentTypeNosniff != "" {
				c.Header("X-Content-Type-Options", cfg.ContentTypeNosniff)
			}
			if cfg.XFrameOptions != "" {
				c.Header("X-Frame-Options", cfg.XFrameOptions)
			}
			if cfg.HSTSMaxAge > 0 {
				value := fmt.Sprintf("max-age=%d", cfg.HSTSMaxAge)
				if cfg.HSTSIncludeSubdomains {
					value += "; includeSubDomains"
				}
				c.Header("Strict-Transport-Security", value)
			}
			if cfg.ContentSecurityPolicy != "" {
				c.Header("Content-Security-Policy", cfg.ContentSecurityPolicy)
			}
			if cfg.ReferrerPolicy != "" {
				c.Header("Referrer-Policy", cfg.ReferrerPolicy)
			}
			return next(c)
		}
	}
}

// SecureDefault returns secure middleware with default settings.
func SecureDefault(next marten.Handler) marten.Handler {
	return Secure(DefaultSecureConfig())(next)
}
