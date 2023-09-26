package isams

import (
	"encoding/json"
	"fmt"
	"time"
)

// fakeUtcZone is the fake time zone for use if one is absent.
const fakeUtcZone = "+00:00"

// Date is a date format which is compatible with the format ISAMS uses. This
// format is a variant on RFC3339 which allows for an absent timezone.
// Essentially, if the timezone is absent, we append the UTC time zone.
type Date time.Time

// UnmarshalJSON unmarshals the given data.
func (i *Date) UnmarshalJSON(data []byte) error {
	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("unmarshal isams date: %w", err)
	}

	// Try parsing normally...
	date, err := time.Parse(time.RFC3339, v)
	if err != nil {
		// Parse with added time zone
		v += fakeUtcZone
		date, err = time.Parse(time.RFC3339, v)
		if err != nil {
			return fmt.Errorf("unmarshal isams date (%s): %w", v, err)
		}
	}

	*i = Date(date)
	return nil
}

// Time is a glorified cast.
func (i *Date) Time() time.Time {
	return time.Time(*i)
}
