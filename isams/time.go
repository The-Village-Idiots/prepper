package isams

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// fakeUtcZone is the fake time zone for use if one is absent.
const (
	fakeUtcZone = "+00:00"
	fakeKitchen = "03:04"
)

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

// Time is a date format which allows for ISAMS period times. These are
// basically Go kitchen times.
type Time time.Time

// NewTime parses and returns a new ISAMS-formatted time object.
func NewTime(src string) (Time, error) {
	segs := strings.Split(src, ":")
	if len(segs) != 2 {
		return Time{}, errors.New("insufficient time segments")
	}

	hour, err := strconv.ParseInt(segs[0], 10, 32)
	if err != nil {
		return Time{}, errors.New("invalid hour syntax")
	}

	suffix := "AM"
	if hour > 12 {
		hour -= 12
		suffix = "PM"
	}

	newsrc := fmt.Sprintf("%d:%s%s", hour, segs[1], suffix)
	date, err := time.Parse(time.Kitchen, newsrc)
	if err != nil {
		return Time{}, err
	}

	return Time(date), nil
}

func (t *Time) UnmarshalJSON(data []byte) error {
	var v string
	if err := json.Unmarshal(data, &v); err != nil {
		return fmt.Errorf("unmarshal isams time: %w", err)
	}

	date, err := NewTime(v)
	if err != nil {
		return fmt.Errorf("unmarshal isams time (%s): %w", v, err)
	}

	*t = Time(date)
	return nil
}
