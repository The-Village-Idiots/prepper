package session

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireAuthentication is a middleware handler which allows a group to be
// protected from unauthorized users. It guarantees that the handler with
// otherwise receive the sessions table unmodified.
type RequireAuthentication struct {
	*Store
	// Should we redirect to login, or show 403?
	Redirect bool
}

// doFail is called if minimum auth requirements are not met.
func (r RequireAuthentication) doFail(c *gin.Context) {
	if r.Redirect {
		c.Redirect(http.StatusFound, "/login")
		c.Abort()

		return
	}

	c.AbortWithStatus(http.StatusForbidden)
}

func (r RequireAuthentication) Handle(c *gin.Context) {
	s := r.Start(c)
	defer s.Update()

	if !s.SignedIn {
		r.doFail(c)
	}
}

// Authenticator returns a new instance of RequireAuthentication ready for use
// with s as the store for cookies.
func Authenticator(s *Store, redirect bool) gin.HandlerFunc {
	return RequireAuthentication{s, redirect}.Handle
}
