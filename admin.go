package main

import (
	"errors"
	"io"
	"net/http"

	"github.com/ejv2/prepper/logging"
	"github.com/gin-gonic/gin"
)

// handleAdminRoot is the handler for "/admin/".
//
// Displays a simple UI for selecting the admin function desired.
func handleAdminRoot(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	c.HTML(http.StatusOK, "admin.gohtml", ddat)
}

func handleAdminLogs(c *gin.Context) {
	l := Dmesg.Grab()
	defer l.Release()

	r := Dmesg.Reader()
	buf, _ := io.ReadAll(r)

	buf = logging.StripColors(buf)
	c.Writer.Write(buf)
}

func handleAdminError(c *gin.Context) {
	internalError(c, errors.New("Admin-Triggered Fatal Error"))
}
