package middleware

import (
	"log"
	"net/http"

	"github.com/gomarten/marten"
)

// Recover catches panics and returns 500.
func Recover(next marten.Handler) marten.Handler {
	return func(c *marten.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("panic recovered: %v", r)
				err = c.Text(http.StatusInternalServerError, "Internal Server Error")
			}
		}()
		return next(c)
	}
}
