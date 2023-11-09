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
	Time     time.Time
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
	if t >= 17 {
		g = "Good evening"
	} else if t >= 12 {
		g = "Good afternoon"
	}

	return DashboardData{u, g, time.Now().Local()}, nil
}

// handleDashboard is the handler for "/dashboard/"
//
// Shows the HTML dashboard page. Route is authenticated by middleware.
func handleDashboard(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	var bk []data.Booking
	var obk []data.Booking
	var dbk []data.Booking
	if ddat.User.IsTechnician() {
		bk, err = data.GetBookings(Database)
		if err != nil {
			internalError(c, err)
			return
		}

		obk, err = data.GetOngoingBookings(Database)
		if err != nil {
			internalError(c, err)
			return
		}

		dbk, err = data.GetBookingsRange(Database, time.Now().Truncate(24*time.Hour), time.Now().Truncate(24*time.Hour).Add(24*time.Hour))
		if err != nil {
			internalError(c, err)
			return
		}
	}

	ubk, err := data.GetPersonalBookingsRange(Database, s.UserID,
		time.Now().Truncate(24*time.Hour), time.Now().Truncate(24*time.Hour).Add(24*time.Hour))
	if err != nil {
		internalError(c, err)
		return
	}

	uuobk, err := data.GetCurrentBooking(Database, s.UserID)
	var uoba *data.Activity
	if err != nil {
		uoba = nil
	} else {
		uuoba := uuobk.Activity.Parent(Database)
		uoba = &uuoba
	}

	dat := struct {
		DashboardData
		// Bookings from this user
		PersonalBookings []data.Booking
		// Bookings from any user
		Bookings []data.Booking
		// Bookings from any user as of today
		DailyBookings []data.Booking
		// Bookings which are currently ongoing
		OngoingBookings []data.Booking
		// Likely current booking
		LikelyCurrent        *data.Activity
		LikelyCurrentBooking data.Booking
	}{ddat, ubk, bk, dbk, obk, uoba, uuobk}

	c.HTML(http.StatusOK, "dashboard.gohtml", dat)
}
