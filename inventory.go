package main

import (
	"fmt"
	"net/http"
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
		tmp := time.Now()
		start = &tmp
	}
	if end == nil {
		tmp := time.Now().Add(1 * time.Hour)
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

	dat := struct {
		DashboardData
		Inventory []AnnotatedItem
	}{ddat, make([]AnnotatedItem, 0, len(e))}

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

// handleItem is the handler for "/inventory/[ITEM]".
//
// Returns an HTML edit page for the given item ID.
func handleItem(c *gin.Context) {
}

func handleItemLocate(c *gin.Context) {
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
		Inventory []AnnotatedItem
	}{ddat, make([]AnnotatedItem, 0, len(e))}

	for _, eq := range e {
		i, err := NewAnnotatedItem(eq)
		if err != nil {
			internalError(c, err)
			return
		}

		dat.Inventory = append(dat.Inventory, i)
	}

	c.HTML(http.StatusOK, "inventory-report.gohtml", dat)
}

func handleInventoryLocate(c *gin.Context) {
}
