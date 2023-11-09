package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ejv2/prepper/data"
	"github.com/ejv2/prepper/isams"
	"github.com/ejv2/prepper/notifications"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Regexes for matching against parameters.
var (
	matchItems = regexp.MustCompile("^qty_.*")
	matchExtra = regexp.MustCompile("^eqty_.*")
)

// Date and time formats for parsing HTML datetime submissions.
const (
	timeFormat = "15:04"
	dateFormat = "2006-01-02"
)

// ItemInformation is the set of information submitted for use in the next
// stage of the booking wizard. Only the item ID, quantity and importance are
// quaranteed to be filled in.
type ItemInformation []data.EquipmentSet

// NewItemInformation parses a new ItemInformation set from a request's query
// parameters.
func NewItemInformation(r *http.Request) (ItemInformation, error) {
	qs := r.URL.Query()

	inf := make(ItemInformation, 0, len(qs))
	for param := range qs {
		if matchItems.MatchString(param) || matchExtra.MatchString(param) {
			segs := strings.Split(param, "_")
			if len(segs) != 2 {
				return inf, fmt.Errorf("parse item information: parse ID: invalid syntax")
			}

			sid := segs[1]
			lid, err := strconv.ParseUint(sid, 10, 32)
			id := uint(lid)
			if err != nil {
				return inf, fmt.Errorf("parse item information: parse ID: %w", err)
			}

			lqty, err := strconv.ParseUint(qs.Get(param), 10, 32)
			qty := uint(lqty)
			if err != nil {
				return inf, fmt.Errorf("parse item information: parse quantity: %w", err)
			}

			inf = append(inf, data.EquipmentSet{
				Quantity:  qty,
				Important: !matchExtra.MatchString(param),
				ItemID:    id,
				Item: data.EquipmentItem{
					Model: &gorm.Model{ID: id},
				},
			})
		}
	}

	return inf, nil
}

// Copy copies any contained items into the destination activity, overwriting
// those already present and adding those which aren't.
func (i ItemInformation) Copy(dest *data.Activity) {
	used := make(map[int]bool, len(i))

	for j, src := range i {
		for k, elem := range dest.Equipment {
			if elem.ItemID == src.ItemID && elem.Important == src.Important {
				used[j] = true
				dest.Equipment[k].Quantity = src.Quantity
				break
			}
		}
	}

	for j, src := range i {
		if !used[j] {
			dest.Equipment = append(dest.Equipment, data.EquipmentSet{
				ActivityID: dest.ID,
				ItemID:     src.ItemID,
				Quantity:   src.Quantity,
				Important:  src.Important,
			})
		}
	}
}

// Next returns the URL of the next stage in the wizard based on this
// information.
func (i ItemInformation) Next(activity uint) string {
	url := url.URL{}
	url.Path = fmt.Sprint("/book/", activity, "/submit")

	q := url.Query()
	for _, item := range i {
		base := "qty_"
		if !item.Important {
			base = "e" + base
		}

		key := fmt.Sprint(base, item.Item.ID)
		q.Add(key, fmt.Sprint(item.Quantity))
	}

	url.RawQuery = q.Encode()
	return url.String()
}

func parseDay(day string) int {
	switch strings.ToLower(day) {
	case "monday":
		return 0
	case "tuesday":
		return 1
	case "wednesday":
		return 2
	case "thursday":
		return 3
	case "friday":
		return 4
	case "saturday":
		return 5
	case "sunday":
		return 6
	}

	// Assume monday if unknown
	return 0
}

// weekCommencing returns the first day of the week which contains the given
// time.
func weekCommencing(target time.Time) time.Time {
	// Week commencing
	wc := target
	// Day of week starts in American style at 0=Sunday
	// We want to start on Monday, so subtract one from the time.
	dow := wc.Weekday()
	if dow == 0 {
		dow = 6
	} else {
		dow--
	}
	// Subtract the number of days equal to the day of the week.
	wc = wc.Add((-24 * time.Hour) * time.Duration(dow))

	return wc
}

// handleBook is the handler for "/book/"
//
// This is the first stage of a multi-step form used for completing a full
// booking.
func handleBook(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	act, err := data.GetPermanentActivities(Database)
	if err != nil {
		internalError(c, err)
		return
	}

	dat := struct {
		DashboardData
		Activities []data.Activity
	}{ddat, act}

	c.HTML(http.StatusOK, "book.gohtml", dat)
}

// handleBookActivity is the handler for "/book/[ACTIVITY_ID]"
//
// This is the second stage of the form.
func handleBookActivity(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	sid := c.Param("activity")
	lid, err := strconv.ParseUint(sid, 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Bad Activity ID")
		return
	}
	id := uint(lid)

	act, err := data.GetActivity(Database, id)
	if err != nil {
		internalError(c, err)
		return
	}

	items, err := data.GetEquipment(Database)
	if err != nil {
		internalError(c, err)
		return
	}

	dat := struct {
		DashboardData
		Activity  data.Activity
		Equipment []data.EquipmentItem
	}{ddat, act, items}
	c.HTML(http.StatusOK, "book-activity.gohtml", dat)
}

// handleBookTimings is the handler for "/book/[ACTIVITY_ID]/timings"
//
// This is the third and final stage of the form and contains the form for
// entering timing and location information. On submission, the booking is
// created and all required rows are copied.
func handleBookTimings(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	sid := c.Param("activity")
	lid, err := strconv.ParseUint(sid, 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Bad Activity ID")
		return
	}
	id := uint(lid)

	act, err := data.GetActivity(Database, id)
	if err != nil {
		internalError(c, err)
		return
	}

	set, err := NewItemInformation(c.Request)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid Paramater Format Format: %s", err)
		return
	}

	var tbl *isams.UserTimetable
	var tbla [][]struct{}
	if Config.HasISAMS() && ddat.User.IsamsID != nil {
		iu, err := ISAMS.FindUser(*ddat.User.IsamsID)
		if err != nil {
			internalError(c, err)
			return
		}

		tbl = iu.Timetable(ISAMS)

		tbla = make([][]struct{}, 0, len(*tbl))
		for _, t := range *tbl {
			tbla = append(tbla, make([]struct{}, t.MaxN()))
		}
	}

	wc := weekCommencing(time.Now())

	dat := struct {
		DashboardData
		Activity       data.Activity
		Items          ItemInformation
		ISAMS          bool
		Timetable      *isams.UserTimetable
		TimetableLoop  [][]struct{}
		WeekCommencing time.Time
	}{ddat, act, set, Config.HasISAMS(), tbl, tbla, wc}
	c.HTML(http.StatusOK, "book-timings.gohtml", dat)
}

func handleBookSubmission(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	usr, err := data.GetUser(Database, s.UserID)
	if err != nil {
		internalError(c, err)
		return
	}

	sid := c.Param("activity")
	lid, err := strconv.ParseUint(sid, 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Bad Activity ID")
		return
	}
	id := uint(lid)

	act, err := data.GetActivity(Database, id)
	if err != nil {
		internalError(c, err)
		return
	}

	set, err := NewItemInformation(c.Request)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid Paramater Format Format: %s", err)
		return
	}

	var date time.Time
	_, manual := c.GetQuery("manual")
	if manual {
		sdate := c.Query("date")
		date, err = time.Parse(dateFormat, sdate)
		if err != nil {
			c.String(http.StatusBadRequest, "Bad Date Format: %s", err.Error())
			return
		}
	} else {
		wcp := c.Query("week_commencing")
		wstart, err := time.Parse(dateFormat, wcp)
		if err != nil {
			c.String(http.StatusBadRequest, "Bad Date Format: %s", err.Error())
			return
		}
		wday := parseDay(c.Query("day"))

		date = weekCommencing(wstart).Truncate(24 * time.Hour).Add((24 * time.Hour) * time.Duration(wday))
	}

	sstime, setime := c.Query("start_time"), c.Query("end_time")
	log.Println(sstime, setime)

	stime, err := time.Parse(timeFormat, sstime)
	if err != nil {
		c.String(http.StatusBadRequest, "Bad Start Time Format: %s", err.Error())
		return
	}

	etime, err := time.Parse(timeFormat, setime)
	if err != nil {
		c.String(http.StatusBadRequest, "Bad End Time Format: %s", err.Error())
		return
	}

	// Handle zone offsets as HTML does not supply them
	_, off := time.Now().Zone()
	stime = stime.Add(time.Duration(off) * -time.Second).In(time.Local).Add(2 * time.Minute)
	etime = etime.Add(time.Duration(off) * -time.Second).In(time.Local).Add(2 * time.Minute)

	// Start/end time is the date + the offset from the day boundary in stime/etime.
	start := date.Add(time.Hour * time.Duration(stime.Hour())).Add(time.Minute * time.Duration(stime.Minute()))
	end := date.Add(time.Hour * time.Duration(etime.Hour())).Add(time.Minute * time.Duration(etime.Minute()))

	location, ok := c.GetQuery("location")
	if !ok {
		c.String(http.StatusBadRequest, "Missing Location Parameter")
		return
	}

	// Copy and clone this activity.
	// Setting extras to nil, as we already appended them earlier.
	set.Copy(&act)
	a, err := act.Clone(Database, s.UserID, nil)
	if err != nil {
		internalError(c, err)
		return
	}

	bk, err := data.NewBooking(Database, a, location, start, end)
	if err != nil {
		internalError(c, err)
		return
	}

	// Push notification out to technicians
	urs, err := data.GetRoleUsers(Database, data.UserTechnician)
	if err == nil {
		// Ignore errors and just push the booking
		for _, u := range urs {
			Notifications.PushUser(u.ID, notifications.Notification{
				Title:  fmt.Sprint("New Booking for ", usr.DisplayName(), " (", usr.Username, ")"),
				Body:   fmt.Sprintln(usr.DisplayName(), "booked", act.Title, "for", bk.StartTime.Format(time.Kitchen), "-", bk.EndTime.Format(time.Kitchen)),
				Type:   notifications.TypeImportant,
				Action: "/tasks/",
				Time:   time.Now(),
			})
		}
	}

	c.Redirect(http.StatusFound, fmt.Sprint("/book/success/", bk.ID))
}

// handleBookSuccess is the handler for "/book/success/[BOOKING_ID]".
func handleBookSuccess(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	sid := c.Param("id")
	lid, err := strconv.ParseUint(sid, 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Bad Booking ID")
		return
	}
	id := uint(lid)

	bk, err := data.GetBooking(Database, id)
	if err != nil {
		if errors.Is(err, data.ErrNoSuchBooking) {
			c.String(http.StatusNotFound, "Booking Not Found")
			return
		}

		internalError(c, err)
		return
	}

	dat := struct {
		DashboardData
		Booking data.Booking
	}{ddat, bk}

	c.HTML(http.StatusOK, "book-complete.gohtml", dat)
}

// handleBooking is the handler for "/book/booking/[ID]".
//
// Shows an HTML summary page for this booking. Intended for use mainly by the
// teacher who created the booking (the owner), unless the user has privileges
// to edit other users' bookings.
func handleBooking(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	sid := c.Param("id")
	lid, err := strconv.ParseUint(sid, 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Bad Booking ID")
		return
	}
	id := uint(lid)

	bk, err := data.GetBooking(Database, id)
	if err != nil {
		if errors.Is(err, data.ErrNoSuchBooking) {
			c.String(http.StatusNotFound, "Booking Not Found")
			return
		}

		internalError(c, err)
		return
	}

	dat := struct {
		DashboardData
		Booking  data.Booking
		Activity data.Activity
	}{ddat, bk, bk.Activity.Parent(Database)}

	c.HTML(http.StatusOK, "booking.gohtml", dat)
}

// handleBookCancel is the handler for "/book/booking/[ID]/cancel".
//
// Marks the given activity as deleted. This does preserve the record in the
// database.
func handleBookCancel(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	sid := c.Param("id")
	lid, err := strconv.ParseUint(sid, 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Bad Booking ID")
		return
	}
	id := uint(lid)

	bk, err := data.GetBooking(Database, id)
	if err != nil {
		if errors.Is(err, data.ErrNoSuchBooking) {
			c.String(http.StatusNotFound, "Booking Not Found")
			return
		}

		internalError(c, err)
		return
	}

	if bk.OwnerID != s.UserID && !ddat.User.Can(data.CapAllBooking) {
		c.String(http.StatusForbidden, "Permission Denied")
		return
	}

	res := Database.Delete(&bk)
	if err := res.Error; err != nil {
		internalError(c, err)
		return
	}

	// Notify technicians of the cancellation.
	urs, err := data.GetRoleUsers(Database, data.UserTechnician)
	if err != nil {
		internalError(c, err)
		return
	}
	for _, usr := range urs {
		Notifications.PushUser(usr.ID, notifications.Notification{
			Title:  "Booking Cancelled",
			Body:   fmt.Sprint(ddat.User.DisplayName(), " (", ddat.User.Username, ") cancelled a booking of ", bk.Activity.Title, " for ", bk.StartTime.Format(time.Kitchen)),
			Action: "/tasks/",
			Type:   notifications.TypeDanger,
			Time:   time.Now(),
		})
	}

	c.Redirect(http.StatusFound, "/dashboard/")
}
