package maintenance

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const defaultErrorMessage = `
Prepper on this server is temporarily unavailable for maintenance. Please check back in 2-5 minutes.

We apologize for any inconvenience.

(If this page shows for an extended period, please contact your system administrator)
`

// defaultError is the default handler to call in case of error.
func defaultError(c *gin.Context) {
	c.AbortWithStatus(http.StatusServiceUnavailable)
	c.String(http.StatusServiceUnavailable, defaultErrorMessage)
}

func Middleware(m *Manager) gin.HandlerFunc {
	return MiddlewareWithHandler(m, defaultError)
}

func MiddlewareWithHandler(m *Manager, onError gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.Is() {
			onError(c)
		}
	}
}
