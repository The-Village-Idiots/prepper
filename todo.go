package main

import (
	"net/http"
	"strconv"

	"github.com/ejv2/prepper/data"
	"github.com/gin-gonic/gin"
)

// handleTodo is the handler for "/todo/".
//
// Returns an HTML page inspired by trello for managing which bookings are in
// progress, unopened and which are completed.
func handleTodo(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	pnd, err := data.GetBookingsStatus(Database, data.BookingStatusPending)
	if err != nil {
		internalError(c, err)
	}

	prog, err := data.GetBookingsStatus(Database, data.BookingStatusProgress)
	if err != nil {
		internalError(c, err)
	}

	done, err := data.GetBookingsStatus(Database, data.BookingStatusReady)
	if err != nil {
		internalError(c, err)
	}

	rej, err := data.GetBookingsStatus(Database, data.BookingStatusRejected)
	if err != nil {
		internalError(c, err)
	}

	dat := struct {
		DashboardData
		Pending  []data.Booking
		Progress []data.Booking
		Done     []data.Booking
		Rejected []data.Booking
	}{ddat, pnd, prog, done, rej}

	c.HTML(http.StatusOK, "todo.gohtml", dat)
}

// handleSetStatus handles promoting the status of a booking given as a URI
// parameter. If there was an error, false is returned, else true.
func handleSetStatus(status data.BookingStatus, c *gin.Context) bool {
	sid := c.Param("id")
	lid, err := strconv.ParseUint(sid, 10, 32)
	id := uint(lid)
	if err != nil {
		c.String(http.StatusBadRequest, "Bad ID Format: %s", err)
		return false
	}

	bk, err := data.GetBooking(Database, id)
	if err != nil {
		internalError(c, err)
		return false
	}

	res := Database.Model(&bk).Where(&bk).Update("Status", status)
	if err := res.Error; err != nil {
		internalError(c, err)
		return false
	}

	return true
}

// handleTodoUnread is the handler for "/todo/unread/[ID]".
func handleTodoUnread(c *gin.Context) {
	if handleSetStatus(data.BookingStatusPending, c) {
		c.Redirect(http.StatusFound, "/todo/")
	}
}

// handleTodoReject is the handler for "/todo/reject/[ID]".
func handleTodoReject(c *gin.Context) {
	if handleSetStatus(data.BookingStatusRejected, c) {
		c.Redirect(http.StatusFound, "/todo/")
	}
}

// handleTodoProgress is the handler for "/todo/progress/[ID]".
func handleTodoProgress(c *gin.Context) {
	if handleSetStatus(data.BookingStatusProgress, c) {
		c.Redirect(http.StatusFound, "/todo/")
	}
}

// handleTodoDone is the handler for "/todo/done/[ID]".
func handleTodoDone(c *gin.Context) {
	if handleSetStatus(data.BookingStatusReady, c) {
		c.Redirect(http.StatusFound, "/todo/")
	}
}
