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

// Booking creation/retrieval errors.
var (
	// ErrNotTemporary is returned when an attempt is made to create a
	// booking directly from a permanent activity. They must be cloned
	// first.
	ErrNotTemporary = errors.New("activity is not temporary")
	// ErrSQL is returned when the operation failed due to an external SQL
	// error.
	ErrSQL = errors.New("sql error")

	// Booking ID is out of range.
	ErrInvalidBookingID = errors.New("invalid booking ID")
	// No booking found with the given ID.
	ErrNoSuchBooking = errors.New("booking does not exist")
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
		StartTime:  start.UTC(),
		EndTime:    end.UTC(),
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

// GetBooking looks up a booking by ID. If the ID is invalid or out of range,
// no such booking exists or an SQL failure is encountered, an error is
// returned.
func GetBooking(db *gorm.DB, id uint) (Booking, error) {
	if id == 0 {
		return Booking{}, fmt.Errorf("get booking %d: %w", id, ErrInvalidBookingID)
	}

	u := Booking{Model: &gorm.Model{ID: id}}
	err := db.Where(&u).Joins("Activity").First(&u).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Booking{}, fmt.Errorf("get booking %d: %w", id, ErrNoSuchBooking)
		}

		return Booking{}, fmt.Errorf("get booking %d: sql error: %w", id, err)
	}

	return u, nil
}
