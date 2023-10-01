package isams

import "time"

// A UserTimetable contains pre-cached timetable information containing all the
// wanted information linked together in one place. A User object's
// UserTimetable is compiled once and is never safe to modify again. This means
// that the expensive linking operations required to set this up are not
// performed until needed and never more than once.
//
// The structure of UserTimetable is as follows:
//
//	Timetable Week (no particular order)
//	  |--> Timetable Day (ordered by day of the week)
//	        |--> Period (ordered by time of day)
//
// This allows you to look up data really quickly with just indirections
// through indexing.
type UserTimetable []StructuredWeek

// StructuredWeek contains an array of StructuredDay(s) and a week name.
type StructuredWeek struct {
	Name string
	Days []StructuredDay
}

// StructuredDay is an array of periods in a day, suitable for sorting using
// sort.Interface.
type StructuredDay struct {
	Name    string
	Periods []StructuredTimetable
}

// Len returns the number of periods in the day.
func (d StructuredDay) Len() int {
	return len(d.Periods)
}

// Less returns true if the period at i starts before the period at j.
func (d StructuredDay) Less(i, j int) bool {
	return d.Periods[i].StartTime.Before(d.Periods[j].StartTime)
}

func (d StructuredDay) Swap(i, j int) {
	tmp := d.Periods[i]

	d.Periods[i] = d.Periods[j]
	d.Periods[j] = tmp
}

type StructuredTimetable struct {
	PeriodCode string

	StartTime time.Time
	EndTime   time.Time

	Room *Classroom
}
