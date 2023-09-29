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

// Building is an entry in the isams EstateManager.
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
