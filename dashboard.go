package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ejv2/prepper/data"
	"github.com/ejv2/prepper/session"
)

type DashboardData struct {
	User     data.User
	Greeting string
}

// NewDashboardData constructs a new DashboardData object for use by the
// frontend from the session s. NewDashboardData panics if the current user is
// invalid (this should never be called on non-authenticated sessions!).
func NewDashboardData(s session.Session) (DashboardData, error) {
	u, err := data.GetUser(Database, s.UserID)
	if err != nil {
		if errors.Is(err, data.ErrUserNotFound) {
			log.Panicf("invalid authenticated session (id %d non-existent)", s.UserID)
		}

		return DashboardData{}, err
	}

	// Greeting is based on time of day
	g := "Good morning"
	t := time.Now().Local().Hour()
	if t >= 12 {
		g = "Good afternoon"
	}

	return DashboardData{u, g}, nil
}

// handleDashboard is the handler for "/dashboard/"
//
// Shows the HTML dashboard page. Route is authenticated by middleware.
func handleDashboard(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	dat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	c.HTML(http.StatusOK, "dashboard.gohtml", dat)
}
