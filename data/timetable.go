package data

import (
	"time"

	"gorm.io/gorm"
)

// A Schedule is an object which is matched to exactly one user. The schedule
// contains a set of Timetables for each timetabled week for the user. This is
// defined by a set of timetabled days (TimetableDay) which contains a set of
// periods matched by foreign key.
type Schedule struct {
	*gorm.Model
	UserID uint

	Weeks []Timetable
}

type Timetable struct {
	*gorm.Model
	ScheduleID uint

	// Ident is the identifying mark for this week. It is most likely to be
	// just "A" or "B".
	Ident rune
	// Days are the set of days (which contains periods) for the current
	// day. There should be exactly seven of these, but missing days will
	// simply be treated as unscheduled.
	Days []TimetableDay
}

type TimetableDay struct {
	*gorm.Model
	TimetableID uint

	// DayOfWeek is the day of the week beginning with Mon (1) and ending
	// with Sun (7).
	DayOfWeek uint8
	// Periods is the periods for the current day.
	Periods []Period
}

type Period struct {
	*gorm.Model
	TimetableDayID uint

	StartTime time.Time
	EndTime   time.Time
	Location  string
	YearGroup uint8
}
