package session

import (
	"log"
	"net/http"

	"github.com/ejv2/prepper/data"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

func (r RequireAuthentication) logReject(c *gin.Context, reason string, uid uint) {
	log.Print("Attempted unauthorized access by UID ", uid, " to ", c.Request.RequestURI, " (", reason, ")")
}

func (r RequireAuthentication) Handle(c *gin.Context) {
	s := r.Start(c)
	defer s.Update()

	if !s.SignedIn {
		r.logReject(c, "not authenticated", s.UserID)
		r.doFail(c)
	}
}

// Authenticator returns a new instance of RequireAuthentication ready for use
// with s as the store for cookies.
func Authenticator(s *Store, redirect bool) gin.HandlerFunc {
	return RequireAuthentication{s, redirect}.Handle
}

// RequirePermissions is a middleware handler which allows a group to be
// protected from unauthorized users and users of insufficient privilege
// levels.
//
// If redirect is true, redirection will occur still on insufficient privilege
// and the users's session will be revoked.
type RequirePermissions struct {
	RequireAuthentication
	// Minimum required permissions.
	Minimum  uint
	Database *gorm.DB
}

func (r RequirePermissions) Handle(c *gin.Context) {
	r.RequireAuthentication.Handle(c)

	s := r.Start(c)
	defer s.Update()

	u, err := data.GetUser(r.Database, s.UserID)
	if err != nil {
		r.logReject(c, "invalid auth token", s.UserID)
		r.doFail(c)
		return
	}

	if !u.Can(uint8(r.Minimum)) {
		r.logReject(c, "insufficient permissions", s.UserID)
		s.Logout()
		r.doFail(c)
		return
	}
}

func Permissions(s *Store, db *gorm.DB, minimum uint, redirect bool) gin.HandlerFunc {
	return RequirePermissions{
		RequireAuthentication: RequireAuthentication{s, redirect},
		Minimum:               minimum,
		Database:              db,
	}.Handle
}
