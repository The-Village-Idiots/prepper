package isams

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"sync"
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

	timetableSetup *sync.Once
	timetable      *UserTimetable
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

// compileTimetable compiles a structured timetable for this user.
func (u *User) compileTimetable(i *ISAMS) {
	log.Println("[iSAMS] Compiling user timetable for", u.UserName)

	sc, err := i.UserSchedule(*u)
	if err != nil {
		return
	}

	// Init storage.
	arr := make(UserTimetable, len(i.weeks))
	for i := range arr {
		arr[i] = make([]StructuredDay, 0, 5)
	}

	// Build initial tree.
	for _, s := range sc {
		for wi, w := range i.weeks {
			for di, d := range w.Days.Day {
				// If day is out of range...
				if di >= len(arr[wi]) {
					// Construct new empty day with capacity for all periods
					arr[wi] = append(arr[wi], make(StructuredDay, 0, len(d.Periods.Period)))
				}

				for _, p := range d.Periods.Period {
					if p.ID == s.PeriodID {
						r, _ := i.FindRoom(uint64(s.RoomID))

						arr[wi][di] = append(arr[wi][di], StructuredTimetable{
							WeekName: &i.weeks[wi].Name,
							DayName:  &i.weeks[wi].Days.Day[di].Name,

							StartTime: time.Time(p.StartTime),
							EndTime:   time.Time(p.EndTime),

							Room:       r,
							PeriodCode: s.Code,
						})
					}
				}
			}
		}
	}

	// Sort leaves in ascending time order.
	for _, w := range arr {
		for _, d := range w {
			sort.Sort(d)
		}
	}

	u.timetable = &arr
	log.Println("[iSAMS] User timetable compilation for", u.UserName, "complete")
}

// Timetable returns this user's structured user timetable. In the common
// case, this returns immediately with cached data. Occasionally, this routine
// needs to build the structured timetable, which can take some time. If nil is
// returned, the timetable should be treated as empty as compilation failed.
func (u *User) Timetable(i *ISAMS) *UserTimetable {
	// We can now guarantee that u.timetable is compiled.
	u.timetableSetup.Do(func() { u.compileTimetable(i) })

	return u.timetable
}

// A Classroom is a possible location for a lesson. Every building on iSAMS has
// zero or more classrooms.
type Classroom struct {
	ID          ID `json:"@Id"`
	Name        string
	Description string
	Initials    string
	Code        string
}

type ClassroomCollection []Classroom

// UnmarshalJSON overrides the unmarshaling routine for this struct, as iSAMS
// likes to pick and choose whether this is a struct or an array.
func (c *ClassroomCollection) UnmarshalJSON(data []byte) error {
	arr := make([]Classroom, 1)

	err := json.Unmarshal(data, &arr)
	if err != nil {
		// Assume an object from now on
		err = json.Unmarshal(data, &arr[0])
		if err != nil {
			return fmt.Errorf("unmarshal isams classroom collection: %w", err)
		}
	}

	*c = ClassroomCollection(arr)
	return nil
}

// Building is an entry in the isams EstateManager. It contains a collection of
// classrooms.
type Building struct {
	ID     ID `json:"@Id"`
	Parent ID

	Name        string
	Description string
	Initials    string

	Classrooms struct {
		Classroom ClassroomCollection
	}
}

// Timetable is a set of Schedule(s) with some associated metadata.
type Timetable struct {
	ID   ID `json:"@Id"`
	Name string

	// Marked as IDs to automatically parse to integers.
	StartYear ID
	EndYear   ID

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

// A Period is a block of time identified by an ID attached to a start and end time.
type Period struct {
	ID        ID `json:"@Id"`
	Name      string
	StartTime Time
	EndTime   Time
}
