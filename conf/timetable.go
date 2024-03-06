package conf

import (
	"encoding/json"
	"fmt"
	"time"
)

// bsearch performs a binary search on the input slice of periods. This is
// implemented recursively so take care on very large timetables.
func bsearch(needle time.Time, stack []*Period) *Period {
	if len(stack) == 0 {
		return nil
	}

	i := len(stack) / 2
	var newstack []*Period

	switch stack[i].Compare(needle) {
	case 0:
		return stack[i]
	case 1:
		// Larger; take upper half
		newstack = stack[i+1:]
	case -1:
		// Smaller; take lower half
		newstack = stack[:i]
	default:
		panic("invalid response from compare")
	}

	return bsearch(needle, newstack)
}

// A TimetableLayout is a set of rules which define how a given day is broken
// up into periods. It is configured losely in order to react to future
// timetable layout changes. Periods must be configured in time order.
type TimetableLayout []*Period

// Returns the first period which is defined as containing this time, or else
// nil if none do.
func (t TimetableLayout) FindPeriod(tm time.Time) *Period {
	return bsearch(tm, t)
}

// A Period is a time which spans from start to end in a given day. A nil
// period is defined as spanning for all of time (such that an empty timetable
// may still be used).
type Period struct {
	Name  string     `json:"name"`
	Start PeriodTime `json:"start"`
	End   PeriodTime `json:"end"`
}

// Within returns true if the given instant in time lies within the period.
// Date information is deliberately discarded, only considering the hour,
// minute and second.
func (p *Period) Within(t time.Time) bool {
	if p == nil {
		return true
	}

	tw := time.Date(0, 1, 1, t.Hour(), t.Minute(), t.Second(), 0, t.Location())
	return (tw.After(time.Time(p.Start)) && tw.Before(time.Time(p.End))) ||
		(tw == time.Time(p.Start)) ||
		(tw == time.Time(p.End))
}

// Compare returns 1 if t lies after this period, -1 if before and zero if
// within.
func (p *Period) Compare(t time.Time) int {
	if p == nil || p.Within(t) {
		return 0
	}

	if t.Before(time.Time(p.Start)) {
		return -1
	}

	return 1
}

// String returns the period's name for implicit formatting.
func (p *Period) String() string {
	if p == nil {
		return "Period 1"
	}

	return p.Name
}

// PeriodTime allows the parsing of human-friendly timing information.
type PeriodTime time.Time

// UnmarshalJSON allows JSON parsing of dates with the desired format.
func (p *PeriodTime) UnmarshalJSON(data []byte) error {
	var tm string
	if err := json.Unmarshal(data, &tm); err != nil {
		return fmt.Errorf("parse period: invalid json: %w", err)
	}

	d, err := time.Parse(time.TimeOnly, tm)
	if err != nil {
		return fmt.Errorf("parse period: %s: invalid time: %w", tm, err)
	}

	*p = PeriodTime(d)
	return nil
}
