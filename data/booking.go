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

type BookingStatus uint

func (b BookingStatus) String() string {
	switch b {
	case BookingStatusPending:
		return "Pending"
	case BookingStatusProgress:
		return "In Progress"
	case BookingStatusReady:
		return "Ready"
	case BookingStatusRejected:
		return "Rejected"
	default:
		return "Unknown"
	}
}

func (b BookingStatus) Pending() bool {
	return b == BookingStatusPending
}

func (b BookingStatus) Progress() bool {
	return b == BookingStatusProgress
}

func (b BookingStatus) Ready() bool {
	return b == BookingStatusReady
}

func (b BookingStatus) Rejected() bool {
	return b == BookingStatusRejected
}

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

	Status BookingStatus

	ActivityID uint
	Activity   Activity

	OwnerID uint
	Owner   User
}

// Past returns true if the end time of the given booking is before the current
// instant in time.
func (b Booking) Past() bool {
	return b.EndTime.Before(time.Now())
}

// Ongoing returns true if the booking is currently ongoing. A booking is
// currently ongoing if the region of time defined by the closed interval over
// [StartTime, EndTime] intersects the the current minute.
func (b Booking) Ongoing() bool {
	t := time.Now().Truncate(time.Minute)
	te := t.Add(time.Minute)

	return (b.StartTime.Before(t) && b.EndTime.After(t)) ||
		(b.StartTime.After(t) && b.StartTime.Before(te))
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
	err := db.Where(&u).Joins("Activity").
		Preload("Activity.Equipment").
		Preload("Activity.Equipment.Item").
		First(&u).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Booking{}, fmt.Errorf("get booking %d: %w", id, ErrNoSuchBooking)
		}

		return Booking{}, fmt.Errorf("get booking %d: sql error: %w", id, err)
	}

	return u, nil
}

// GetBookings returns all relevant bookings from the database. Relevant
// bookings are those with bookings dates greater than the current time.
func GetBookings(db *gorm.DB) ([]Booking, error) {
	b := make([]Booking, 0, 5)
	res := db.Model(&Booking{}).Joins("Activity").Joins("Owner").
		Where("start_time > ?", time.Now()).
		Find(&b)

	if err := res.Error; err != nil {
		return b, fmt.Errorf("get bookings: sql error: %s", err)
	}

	return b, nil
}

// GetPersonalBookings further filters down relevant bookings to those which
// are owned by the given ID.
func GetPersonalBookings(db *gorm.DB, id uint) ([]Booking, error) {
	if id == 0 {
		return nil, fmt.Errorf("get personal bookings for %d: %w", id, ErrInvalidID)
	}

	b := make([]Booking, 0, 5)
	res := db.Model(&Booking{}).Joins("Activity").Joins("Owner").
		Where("start_time > ?", time.Now()).
		Where(&Booking{OwnerID: id}).
		Find(&b)

	if err := res.Error; err != nil {
		return b, fmt.Errorf("get personal bookings for %d: sql error: %w", id, err)
	}

	return b, nil
}

// GetBookingsRange returns all relevant bookings from the database which fall
// within the given timeframe. A booking is defined as within the current
// timeframe if any part of its booked period intersects with the period
// defined by the closed range start to end. Relevant bookings are those with
// bookings dates greater than the current time.
func GetBookingsRange(db *gorm.DB, start, end time.Time) ([]Booking, error) {
	b := make([]Booking, 0, 5)
	res := db.Model(&Booking{}).Joins("Activity").Joins("Owner").
		Where(`
			(start_time <= ? AND end_time >= ?) OR
			(start_time >= ? AND start_time <= ?)
		 `, start, start, start, end).
		Find(&b)

	if err := res.Error; err != nil {
		return b, fmt.Errorf("get bookings: sql error: %s", err)
	}

	return b, nil
}

// GetPersonalBookingsRange filters bookings in the given range to those booked
// by the given ID.
func GetPersonalBookingsRange(db *gorm.DB, uid uint, start, end time.Time) ([]Booking, error) {
	b := make([]Booking, 0, 5)
	res := db.Model(&Booking{}).Joins("Activity").Joins("Owner").
		Where(&Booking{OwnerID: uid}).
		Where(`
			(start_time <= ? AND end_time >= ?) OR
			(start_time >= ? AND start_time <= ?)
		 `, start, start, start, end).
		Find(&b)

	if err := res.Error; err != nil {
		return b, fmt.Errorf("get bookings: sql error: %s", err)
	}

	return b, nil
}

// GetBookingsStatus returns all bookings of the given status.
func GetBookingsStatus(db *gorm.DB, status BookingStatus) ([]Booking, error) {
	b := make([]Booking, 0, 5)
	res := db.Model(&Booking{}).Joins("Activity").Joins("Owner").
		Where(&Booking{Status: status}).
		Find(&b)

	if err := res.Error; err != nil {
		return b, fmt.Errorf("get %s bookings: sql error: %w", status, err)
	}

	return b, nil
}

// GetOngoingBookings returns any bookings which are currently ongoing. These
// are defined as bookings which intersect with the current minute.
func GetOngoingBookings(db *gorm.DB) ([]Booking, error) {
	return GetBookingsRange(db, time.Now().Truncate(time.Minute), time.Now().Truncate(time.Minute).Add(time.Minute))
}

// GetPersonalBookings filters ongoing bookings to those booked by the given
// user ID.
func GetPersonalOngoingBookings(db *gorm.DB, uid uint) ([]Booking, error) {
	return GetPersonalBookingsRange(db, uid, time.Now().Truncate(time.Minute), time.Now().Truncate(time.Minute).Add(time.Minute))
}

// GetCurrentBooking gets the most likely currently ongoing booking for the
// given user.
func GetCurrentBooking(db *gorm.DB, uid uint) (Booking, error) {
	o, err := GetPersonalOngoingBookings(db, uid)
	if err != nil {
		return Booking{}, err
	}

	if len(o) == 0 {
		return Booking{}, fmt.Errorf("get current booking for %v: nothing appropriate", uid)
	}

	return o[0], nil
}
