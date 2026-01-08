package middleware

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gomarten/marten"
)

// BasicAuthConfig configures basic authentication.
type BasicAuthConfig struct {
	Realm    string
	Validate func(user, pass string) bool
}

// BasicAuth returns a basic authentication middleware.
func BasicAuth(cfg BasicAuthConfig) marten.Middleware {
	if cfg.Realm == "" {
		cfg.Realm = "Restricted"
	}

	return func(next marten.Handler) marten.Handler {
		return func(c *marten.Ctx) error {
			auth := c.Request.Header.Get("Authorization")
			if auth == "" || !strings.HasPrefix(auth, "Basic ") {
				return unauthorized(c, cfg.Realm)
			}

			payload, err := base64.StdEncoding.DecodeString(auth[6:])
			if err != nil {
				return unauthorized(c, cfg.Realm)
			}

			pair := strings.SplitN(string(payload), ":", 2)
			if len(pair) != 2 || !cfg.Validate(pair[0], pair[1]) {
				return unauthorized(c, cfg.Realm)
			}

			c.Set("user", pair[0])
			return next(c)
		}
	}
}

// BasicAuthSimple creates a basic auth middleware with a single user/pass.
func BasicAuthSimple(user, pass string) marten.Middleware {
	return BasicAuth(BasicAuthConfig{
		Validate: func(u, p string) bool {
			return subtle.ConstantTimeCompare([]byte(u), []byte(user)) == 1 &&
				subtle.ConstantTimeCompare([]byte(p), []byte(pass)) == 1
		},
	})
}

func unauthorized(c *marten.Ctx, realm string) error {
	c.Header("WWW-Authenticate", `Basic realm="`+realm+`"`)
	return c.JSON(http.StatusUnauthorized, marten.E("unauthorized"))
}
