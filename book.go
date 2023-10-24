package main

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/ejv2/prepper/data"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Regexes for matching against parameters.
var (
	matchItems = regexp.MustCompile("^qty_.*")
	matchExtra = regexp.MustCompile("^eqty_.*")
)

// ItemInformation is the set of information submitted for use in the next
// stage of the booking wizard. Only the item ID, quantity and importance are
// quaranteed to be filled in.
type ItemInformation []data.EquipmentSet

// NewItemInformation parses a new ItemInformation set from a request's query
// parameters.
func NewItemInformation(r *http.Request) (ItemInformation, error) {
	qs := r.URL.Query()

	inf := make(ItemInformation, len(qs))
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

// Next returns the URL of the next stage in the wizard based on this
// information.
func (i ItemInformation) Next(activity uint) string {
	url := url.URL{}
	url.Path = fmt.Sprint("/book/", activity, "/submit")

	for _, item := range i {
		base := "qty_"
		if !item.Important {
			base = "e" + base
		}

		key := fmt.Sprint(base, item.Item.ID)
		url.Query().Add(key, fmt.Sprint(item.Quantity))
	}

	return url.String()
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
	}

	dat := struct {
		DashboardData
		Activity data.Activity
		Items    ItemInformation
	}{ddat, act, set}
	c.HTML(http.StatusOK, "book-timings.gohtml", dat)
}
