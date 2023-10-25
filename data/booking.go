package data

import (
	"errors"
	"fmt"
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

// Booking creation errors.
var (
	// ErrNotTemporary is returned when an attempt is made to create a
	// booking directly from a permanent activity. They must be cloned
	// first.
	ErrNotTemporary = errors.New("activity is not temporary")
	// ErrSQL is returned when the operation failed due to an external SQL
	// error.
	ErrSQL = errors.New("sql error")
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

// NewBooking inserts a new booking from the specified activity into the
// database. The owner is taken to be the owner of act. If act is not yet a
// temporary activity, an error is returned. Else, all errors returned will be
// SQL-related.
func NewBooking(db *gorm.DB, act Activity, location string, start, end time.Time) (Booking, error) {
	if !act.Temporary {
		return Booking{}, fmt.Errorf("book activity %s: %w", act.Title, ErrNotTemporary)
	}

	bk := Booking{
		StartTime:  start.Local(),
		EndTime:    end.Local(),
		Location:   location,
		ActivityID: act.ID,
		OwnerID:    act.OwnerID,
	}

	err := db.Create(&bk).Error
	if err != nil {
		return bk, fmt.Errorf("book activity %s: %w", act.Title, ErrSQL)
	}

	return bk, nil
}
