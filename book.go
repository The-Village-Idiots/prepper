package main

import (
	"net/http"
	"strconv"

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

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	act, err := data.GetPermanentActivities(Database)
	if err != nil {
		internalError(c, err)
		return
	}

	dat := struct {
		DashboardData
		Activities []data.Activity
	}{ddat, act}

	c.HTML(http.StatusOK, "book.gohtml", dat)
}

// handleBookActivity is the handler for "/book/[ACTIVITY_ID]"
//
// This is the second stage of the form.
func handleBookActivity(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	sid := c.Param("activity")
	lid, err := strconv.ParseUint(sid, 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Bad Activity ID")
		return
	}
	id := uint(lid)

	act, err := data.GetActivity(Database, id)
	if err != nil {
		internalError(c, err)
		return
	}

	items, err := data.GetEquipment(Database)
	if err != nil {
		internalError(c, err)
		return
	}

	dat := struct {
		DashboardData
		Activity  data.Activity
		Equipment []data.EquipmentItem
	}{ddat, act, items}
	c.HTML(http.StatusOK, "book-activity.gohtml", dat)
}

// handleBookTimings is the handler for "/book/[ACTIVITY_ID]/timings"
//
// This is the third and final stage of the form and contains the form for
// entering timing and location information. On submission, the booking is
// created and all required rows are copied.
func handleBookTimings(c *gin.Context) {
}
