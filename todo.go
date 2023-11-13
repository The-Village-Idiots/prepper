package main

import (
	"net/http"

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
