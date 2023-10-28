package main

import (
	"io"
	"net/http"

	"github.com/ejv2/prepper/logging"
	"github.com/gin-gonic/gin"
)

// handleAdminRoot is the handler for "/admin/".
//
// This is mainly used to check admin permissions and is not particularly
// useful. This route always returns 200 OK for authenticated users but with no
// response.
func handleAdminRoot(c *gin.Context) {
	c.String(http.StatusOK, "Hello!")
}

func handleAdminLogs(c *gin.Context) {
	l := Dmesg.Grab()
	defer l.Release()

	r := Dmesg.Reader()
	buf, _ := io.ReadAll(r)

	buf = logging.StripColors(buf)
	c.Writer.Write(buf)
}
