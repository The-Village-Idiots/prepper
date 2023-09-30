package isams

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ISAMS request detail constants.
const (
	apiEndpoint = "/api/batch/1.0/json.ashx?apiKey="
)

// ISAMS usage errors.
var (
	ErrInit        = errors.New("not initialized")
	ErrRequest     = errors.New("request failed")
	ErrRequestIO   = errors.New("request I/O failed")
	ErrEncoding    = errors.New("bad response encoding")
	ErrPrepopulate = errors.New("prepopulation failed")

	ErrNotFound = errors.New("not found")
	ErrEmptySet = errors.New("nothing matching found")
)

// isamsEndpoint is a configured requestable endpoint, formatted to request the
// correct API via HTTPS.
type isamsEndpoint struct {
	domain, key string
}

func (i isamsEndpoint) String() string {
	return fmt.Sprintf("https://%s%s{%s}", i.domain, apiEndpoint, i.key)
}

// ISAMS is the data manager for a live ISAMS connection. Any exported fields
// are pre-populated at startup and are not written to again (and as such
// should not be modified).
type ISAMS struct {
	endpoint isamsEndpoint
	client   *http.Client

	Users []User
	Rooms []Classroom

	CurrentTimetable Timetable
	weeks            []TimetableWeek
}

func (i *ISAMS) request() (*isamsResponse, error) {
	if i.client == nil {
		return nil, ErrInit
	}

	resp, err := i.client.Get(i.endpoint.String())
	if err != nil {
		return nil, fmt.Errorf("isams at %s: %w: %s", i.endpoint.String(), ErrRequest, err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("isams at %s: %w: response code %d", i.endpoint.String(), ErrRequest, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("isams at %s: %w: %s", i.endpoint.String(), ErrRequestIO, err.Error())
	}
	defer resp.Body.Close()

	// NOTE: The structure of this struct is really important. DO NOT CHANGE!
	obj := struct {
		I isamsResponse `json:"iSAMS"`
	}{}

	err = json.Unmarshal(body, &obj)
	if err != nil {
		return nil, fmt.Errorf("isams at %s: %w: %s", i.endpoint.String(), ErrEncoding, err.Error())
	}

	return &obj.I, nil
}

// prepopulate loads startup data from the database to prepopulate exported
// fields.
func (i *ISAMS) prepopulate() error {
	resp, err := i.request()
	if err != nil {
		return err
	}

	i.Users = resp.HRManager.CurrentStaff.StaffMember
	i.CurrentTimetable = resp.TimetableManager.PublishedTimetables.Timetable
	i.weeks = resp.TimetableManager.Structure.Week

	// Wild guess at how many rooms we might have
	// Maybe about 5 classrooms per building?
	i.Rooms = make([]Classroom, 0, len(resp.EstateManager.Buildings.Building)*5)
	for _, b := range resp.EstateManager.Buildings.Building {
		i.Rooms = append(i.Rooms, b.Classrooms.Classroom...)
	}

	return nil
}

// validateTimetable checks if the currently published timetable is marked for
// this year. If not, return false.
func (i *ISAMS) validateTimetable() bool {
	t := time.Now()
	yy := ID(t.Year())

	if yy > i.CurrentTimetable.EndYear || yy < i.CurrentTimetable.StartYear {
		return false
	}

	return true
}

// FindUser looks up a user by usercode, which should be the ID used across the
// rest of the codebase to uniquely identify a user (due to it being used
// across the API).
func (i *ISAMS) FindUser(ucode string) (*User, error) {
	for _, u := range i.Users {
		if u.UserCode == ucode {
			return &u, nil
		}

	}

	return nil, fmt.Errorf("find isams user %s: %w", ucode, ErrNotFound)
}

// FindRoom looks up a classroom by room ID. Rooms are not designed for storage
// for reference (as looking them up is kind of expensive), so it is instead
// recommended to copy out details of the room where they are required.
func (i *ISAMS) FindRoom(id uint64) (*Classroom, error) {
	for _, r := range i.Rooms {
		if r.ID == ID(id) {
			return &r, nil
		}
	}

	return nil, fmt.Errorf("find isams room ID %d: %w", id, ErrNotFound)
}

// UserSchedule returns matching schedule objects for this user. These are in
// no particular order and further investigation is needed to obtain period and
// timing information.
//
// TODO: This is in some need of heavy optimization! Perhaps a caching hash map
// against user codes with pointers to correct periods to save space?
func (i *ISAMS) UserSchedule(u User) ([]Schedule, error) {
	// Assuming a full timetable, perhaps about 7*5 periods?
	arr := make([]Schedule, 0, 35)
	for _, s := range i.CurrentTimetable.Schedules.Schedule {
		if s.Teacher == u.UserCode {
			arr = append(arr, s)
		}
	}

	if len(arr) == 0 {
		return nil, fmt.Errorf("find user schedule for %s: %w", u.UserName, ErrEmptySet)
	}

	return arr, nil
}

// Period looks up a period from its ID, returning an error if none is found.
//
// TODO: This also needs A LOT of optimization!
func (i *ISAMS) Period(id uint64) (Period, error) {
	for _, w := range i.weeks {
		for _, d := range w.Days.Day {
			for _, p := range d.Periods.Period {
				if p.ID == ID(id) {
					return p, nil
				}
			}
		}
	}

	return Period{}, ErrNotFound
}

// SchedulePeriod looks up a period from the associated period of a Schedule
// entry.
func (i *ISAMS) SchedulePeriod(s Schedule) (Period, error) {
	return i.Period(uint64(s.PeriodID))
}

// New loads a new intance of the ISAMS data manager. The first action is to
// reach out to the ISAMS database to retrieve basic information about
// timetables etc. This also verifies that the connection has been set up
// correctly.
func New(domain, key string) (*ISAMS, error) {
	obj := &ISAMS{
		isamsEndpoint{domain, key},
		&http.Client{},

		[]User{},
		[]Classroom{},
		Timetable{},
		[]TimetableWeek{},
	}

	if err := obj.prepopulate(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrPrepopulate, err)
	}

	if !obj.validateTimetable() {
		return nil, fmt.Errorf("%w: found no valid published timetable", ErrPrepopulate)
	}

	return obj, nil
}
