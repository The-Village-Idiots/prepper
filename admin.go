package main

import (
	"errors"
	"io"
	"log"
	"net/http"
	"time"

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

func handleAdminMaintenance(c *gin.Context) {
	if Maintenance.Enter() != nil {
		log.Panic("somehow got to admin maintenance handler when in maintenance?")
	}

	log.Println(c.RemoteIP(), "enables maintenance mode from admin panel")
	c.String(http.StatusOK, "Maintenance mode enabled by system administrator.\nTimestamp: %v", time.Now().Format(time.RFC1123))
}

// handleAdminRunMaint is the handler for "/admin/runnow".
//
// Runs admin maintenance tasks now!
func handleAdminRunMaint(c *gin.Context) {
	MSched.Now()
	c.Redirect(http.StatusFound, "/admin/")
}
