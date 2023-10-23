package main

import (
	"net/http"

	"github.com/ejv2/prepper/data"
	"github.com/gin-gonic/gin"
)

// handleBook is the handler for "/book/"
//
// This is the first stage of a multi-step form used for completing a full
// booking.
func handleBook(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	dat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	act, err := data.GetPermanentActivities(Database)
	if err != nil {
		internalError(c, err)
		return
	}

	ddat := struct {
		DashboardData
		Activities []data.Activity
	}{dat, act}

	c.HTML(http.StatusOK, "book.gohtml", ddat)
}

// handleBookActivity is the handler for "/book/[ACTIVITY_ID]"
//
// This is the second stage of the form.
func handleBookActivity(c *gin.Context) {
}
