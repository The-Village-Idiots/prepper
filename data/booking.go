package data

import (
	"time"

	"gorm.io/gorm"
)

// Booking status.
//
// Every booking has an associated status which may be changed at will by the
// technicians (and to certain values by the owning teacher account). These are
// used to visually communicate the status of certain tasks between users.
const (
	// Pending review by a lab technician. Not yet acknowledged.
	BookingStatusPending = iota
	// In progress. The lab technician has viewed the request and is
	// processing it.
	BookingStatusProgress
	// Ready. This booking has been prepared and is ready for review.
	BookingStatusReady
	// Booking rejected. The technician is unable to fulfil this request
	// and has rejected it. May be accompanied by a rejection message.
	BookingStatusRejected
)

// A Booking is an entry in the schedule which has an associated activity
// (temporary or persistent), teacher account and location specification. The
// location specification stores a location as given by the timetable or other
// data source (manually, iSAMS data, etc.). A Booking also has an associated
// status which is documented above.
type Booking struct {
	*gorm.Model

	StartTime time.Time
	EndTime   time.Time
	Location  string

	Status uint

	ActivityID uint
	Activity   Activity

	OwnerID uint
	Owner   User
}
