package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ejv2/prepper/data"
	"github.com/gin-gonic/gin"
)

// AnnotatedInventory contains an item from the database, as well as various
// pieces of information about it which may be useful for client pages.
type AnnotatedItem struct {
	data.EquipmentItem

	Bookings      []data.Booking
	DailyBookings []data.Booking

	Use      int
	DailyUse int
	Balance  int
}

// DailyBooked returns the total number of items booked for the current day.
func (a AnnotatedItem) DailyBooked() int {
	count := 0
	for _, b := range a.DailyBookings {
		for _, e := range b.Activity.Equipment {
			if e.ItemID == a.ID {
				count += int(e.Quantity)
			}
		}
	}

	return count
}

// NewAnnotatedItemTime returns a new AnnotatedItem from the given equipment
// item. NewAnnotatedItem uses the passed start and end times. If start or end
// ar nil, the current time is used (+- 1 hour).
func NewAnnotatedItemTime(i data.EquipmentItem, start, end *time.Time) (an AnnotatedItem, err error) {
	an = AnnotatedItem{EquipmentItem: i}

	// If start/end is nil, fill in default times.
	if start == nil {
		tmp := time.Now().Truncate(time.Minute)
		start = &tmp
	}
	if end == nil {
		tmp := time.Now().Truncate(time.Minute).Add(time.Minute)
		end = &tmp
	}

	an.Bookings, err = i.Bookings(*start, *end)
	if err != nil {
		return an, fmt.Errorf("new annotated item: %w", err)
	}

	an.Use, err = i.Usage(*start, *end)
	if err != nil {
		return an, fmt.Errorf("new annotated item: %w", err)
	}

	an.Balance, err = i.NetQuantity(*start, *end)
	if err != nil {
		return an, fmt.Errorf("new annotated item: %w", err)
	}

	an.DailyBookings, err = i.DailyBookings(time.Now())
	if err != nil {
		return an, fmt.Errorf("new annotated item: %w", err)
	}
	an.DailyUse, err = i.DailyUsage(time.Now())
	if err != nil {
		return an, fmt.Errorf("new annotated item: %w", err)
	}

	return
}

// NewAnnotatedItem calls NewAnnotatedItemTime assuming for the current day.
func NewAnnotatedItem(i data.EquipmentItem) (AnnotatedItem, error) {
	return NewAnnotatedItemTime(i, nil, nil)
}

// handleInventory is the handler for "/inventory/".
func handleInventory(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	e, err := data.GetEquipment(Database)
	if err != nil {
		internalError(c, err)
		return
	}

	dname, del := c.GetQuery("deleted")

	dat := struct {
		DashboardData
		Inventory   []AnnotatedItem
		Deleted     bool
		DeletedName string
	}{ddat, make([]AnnotatedItem, 0, len(e)), del, dname}

	for _, eq := range e {
		i, err := NewAnnotatedItem(eq)
		if err != nil {
			internalError(c, err)
			return
		}

		dat.Inventory = append(dat.Inventory, i)
	}

	c.HTML(http.StatusOK, "inventory.gohtml", dat)
}

// handleItem is the handler for "/inventory/item/[ITEM]".
//
// Returns an HTML edit page for the given item ID.
func handleItem(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	siid := c.Param("id")
	lid, err := strconv.ParseUint(siid, 10, 32)
	id := uint(lid)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid Item ID")
		return
	}

	item, err := data.GetEquipmentItem(Database, id)
	if err != nil {
		c.String(http.StatusNotFound, "Item Not Found")
		return
	}

	aitem, err := NewAnnotatedItem(item)
	if err != nil {
		internalError(c, err)
		return
	}

	dat := struct {
		DashboardData
		Item AnnotatedItem
	}{ddat, aitem}

	c.HTML(http.StatusOK, "item.gohtml", dat)
}

func handleItemLocate(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	siid := c.Param("id")
	lid, err := strconv.ParseUint(siid, 10, 32)
	id := uint(lid)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid Item ID")
		return
	}

	item, err := data.GetEquipmentItem(Database, id)
	if err != nil {
		c.String(http.StatusNotFound, "Item Not Found")
		return
	}

	aitem, err := NewAnnotatedItem(item)
	if err != nil {
		internalError(c, err)
		return
	}

	dat := struct {
		DashboardData
		Item         AnnotatedItem
		Bookings     []data.Booking
		PastBookings []data.Booking
	}{ddat, aitem, []data.Booking{}, []data.Booking{}}

	dayStart := time.Now().Local().Truncate(24 * time.Hour)
	minuteStart := time.Now().Local().Truncate(time.Minute)
	minuteEnd := minuteStart.Add(time.Minute)

	dat.Bookings, err = item.Bookings(minuteStart, minuteEnd)
	if err != nil {
		internalError(c, err)
		return
	}

	dat.PastBookings, err = item.Bookings(dayStart, time.Now().Local())
	if err != nil {
		internalError(c, err)
		return
	}

	c.HTML(http.StatusOK, "item-find.gohtml", dat)
}

// handleNewItem is the handler for "/inventory/new".
//
// Returns an HTML page with some JavaScript forms for creating new items in
// bulk. Item creation itself is handled via the API.
func handleNewItem(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	dat := struct {
		DashboardData
		Inventory []data.EquipmentItem
	}{ddat, nil}

	c.HTML(http.StatusOK, "inventory-add.gohtml", dat)
}

func handleInventoryReport(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	e, err := data.GetEquipment(Database)
	if err != nil {
		internalError(c, err)
		return
	}

	dat := struct {
		DashboardData
		Inventory        []AnnotatedItem
		PreviouslyBooked []AnnotatedItem
	}{ddat, make([]AnnotatedItem, 0, len(e)), make([]AnnotatedItem, 0, len(e))}

	for _, eq := range e {
		i, err := NewAnnotatedItem(eq)
		if err != nil {
			internalError(c, err)
			return
		}

		dat.Inventory = append(dat.Inventory, i)

		today := time.Now().Local().Truncate(24 * time.Hour)
		yesterday := today.Add(-24 * time.Hour)

		old, err := eq.Bookings(yesterday, today)
		if err == nil && len(old) != 0 {
			iold, err := NewAnnotatedItemTime(eq, &yesterday, &today)
			if err == nil {
				dat.PreviouslyBooked = append(dat.PreviouslyBooked, iold)
			}
		}
	}

	c.HTML(http.StatusOK, "inventory-report.gohtml", dat)
}

func handleInventoryLocate(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	siid, ok := c.GetQuery("item")
	if ok {
		iid, err := strconv.ParseUint(siid, 10, 32)
		if err != nil {
			c.Redirect(http.StatusFound, "/inventory/locate?error")
			return
		}

		_, err = data.GetEquipmentItem(Database, uint(iid))
		if err != nil {
			c.Redirect(http.StatusFound, "/inventory/locate?error")
			return
		}

		c.Redirect(http.StatusFound, fmt.Sprint("/inventory/item/", iid, "/locate"))
		return
	}

	_, haserr := c.GetQuery("error")

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	e, err := data.GetEquipment(Database)
	if err != nil {
		internalError(c, err)
		return
	}

	dat := struct {
		DashboardData
		Inventory []AnnotatedItem
		Error     bool
	}{ddat, make([]AnnotatedItem, 0, len(e)), haserr}

	for _, eq := range e {
		i, err := NewAnnotatedItem(eq)
		if err != nil {
			internalError(c, err)
			return
		}

		dat.Inventory = append(dat.Inventory, i)
	}

	c.HTML(http.StatusOK, "inventory-find.gohtml", dat)
}

// handleItemDelete is the handler for "/inventory/item/[ID]/delete"
//
// Marks the item with the ID given in the URI parameter
// as deleted.
func handleItemDelete(c *gin.Context) {
	s := Sessions.Start(c)
	defer s.Update()

	ddat, err := NewDashboardData(s)
	if err != nil {
		internalError(c, err)
		return
	}

	if !ddat.User.Can(data.CapManageOtherInventory) {
		c.String(http.StatusForbidden, "Access Denied")
		return
	}

	sid := c.Param("id")
	lid, err := strconv.ParseUint(sid, 10, 32)
	id := uint(lid)
	if err != nil {
		c.String(http.StatusBadRequest, "Bad Item ID")
		return
	}

	it, err := data.GetEquipmentItem(Database, id)
	if err != nil {
		c.String(http.StatusNotFound, "Item Not Found")
		return
	}

	res := Database.Delete(&it)
	if res.Error != nil {
		internalError(c, err)
		return
	}

	ename := url.QueryEscape(it.Name)
	c.Redirect(http.StatusFound, "/inventory/?deleted="+ename)
}
