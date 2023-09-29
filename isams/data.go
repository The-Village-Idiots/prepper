package isams

import (
	"time"

	"github.com/ejv2/prepper/data"
)

// This object is massive (and a mess). Please don't copy it!
//
// NOTE: This object is messy as it necessarily reflects the ISAMS API design.
type isamsResponse struct {
	HRManager struct {
		CurrentStaff struct {
			StaffMember []User
		}
	}

	EstateManager struct {
		Buildings struct {
			Building []Building
		}
	}

	TimetableManager struct {
		PublishedTimetables struct {
			Timetable Timetable
		}
		Structure struct {
			Week []TimetableWeek
		}
	}
}

// User is the definitions for the isams user object returned from the bulk
// API. Fields which do not have a JSON field tag *must not* have their names
// tampered with as they are designed to coincide with the API JSON format.
type User struct {
	// IDs (yes all three of them).
	ID         ID     `json:"@Id"`
	PersonID   ID     `json:"@PersonId"`
	PersonGUID string `json:"@PersonGuid"`

	// Identification.
	UserName string
	UserCode string

	// Smalltalk section.
	Title      string
	Forename   string
	Surname    string
	Salutation string

	// Contact details.
	SchoolEmailAddress string
	SchoolMobileNumber string

	// Used to filter out old records.
	LeavingDate *Date
}

// StillHere checks if the leaving date of a record is either nil or after now.
func (u User) StillHere() bool {
	return u.LeavingDate == nil || u.LeavingDate.Time().After(time.Now())
}

// DataUser returns an equivalent data.User for this user object.
func (u User) DataUser() data.User {
	return data.User{
		FirstName: u.Forename,
		LastName:  u.Surname,
		Title:     u.Title,

		Email:     u.SchoolEmailAddress,
		Telephone: u.SchoolMobileNumber,

		IsamsID: &u.UserCode,
	}
}

// A Classroom is a possible location for a lesson. Every building on iSAMS has
// zero or more classrooms.
type Classroom struct {
	ID          ID `json:"~Id"`
	Name        string
	Description string
	Initials    string
	Code        string
}

// Building is an entry in the isams EstateManager. It contains a collection of
// classrooms.
type Building struct {
	ID     ID `json:"@Id"`
	Parent ID

	Name        string
	Description string
	Initials    string

	Classroom Bool

	Classrooms struct {
		Classroom []Classroom
	}
}

// Timetable is a set of Schedule(s) with some associated metadata.
type Timetable struct {
	ID        ID `json:"@Id"`
	Name      string
	StartYear string
	EndYear   string

	Schedules struct {
		Schedule []Schedule
	}
}

// Schedule is an entry in the timetable. It contains a reference to a Period
// (which gives us timing information) along with a reference to the assigned
// teacher and room. The SetID and Set are always set to either 1 or zero, for
// some reason.
type Schedule struct {
	ID   ID `json:"@Id"`
	Code string

	// This is the UserCode of the teacher.
	Teacher string
	// This is the ID in the classroom table.
	RoomID   ID `json:"RoomId"`
	PeriodID ID `json:"PeriodId"`
}

// A TimetableWeek contains a collection of timetabled days (of which there
// must be at least 5), along with some metadata. We discard divisions
// information here as it isn't particularly important to us.
type TimetableWeek struct {
	ID        ID `json:"@Id"`
	Name      string
	ShortName string
	Ordinal   ID
	Active    Bool

	Days struct {
		Day []TimetableDay
	}
}

// A TimetableDay is a component of a TimetableWeek which contains one or more
// periods.
type TimetableDay struct {
	ID        ID `json:"@Id"`
	Name      string
	ShortName string
	Day       ID
	Ordinal   ID
	Active    Bool

	Periods struct {
		Period []Period
	}
}

// A Period is a block of time identified by an ID attached to a start and end
// time.
type Period struct {
	ID        ID `json:"@Id"`
	Name      string
	StartTime Time
	EndTime   Time
}
