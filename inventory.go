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
