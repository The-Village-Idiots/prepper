package data

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// An EquipmentItem is an entry in the inventory. It has an associated stock
// level and some flags to show the status of the equipment (such as any
// warnings for hazards etc).
type EquipmentItem struct {
	*gorm.Model

	Name        string
	Description string

	Quantity uint
	// Availability override. If false, quantity is treated as though zero.
	Available bool

	HazardVoltage bool
	HazardToxic   bool
	HazardLazer   bool
	HazardMisc    bool

	// For convenience.
	db *gorm.DB
}

// Bookings returns any bookings which use this item between the given start
// and end time.
func (e *EquipmentItem) Bookings(start, end time.Time) ([]Booking, error) {
	if e.db == nil {
		panic("use of db operation on non-selected equipment")
	}

	// b are bookings in the time range.
	b := make([]Booking, 0, 5)

	// Possible timing overlaps are:
	//    ---MATCH---
	//         -----REGION----
	//           ---MATCH---
	//                     ---MATCH---
	res := e.db.Model(&Booking{}).Joins("Activity").
		Where("(? >= start_time AND ? <= end_time) OR (? <= start_time AND ? >= start_time AND ? <= end_time) OR (? <= end_time AND ? >= end_time)",
			start, end,
			start, end, end,
			start, end).
		Preload("Activity.Equipment").
		Preload("Activity.Equipment.Item").
		Find(&b)

	if res.Error != nil {
		return b, fmt.Errorf("%s bookings (%v until %v): %w", e.Name, start, end, res.Error)
	}

	bm := make([]Booking, 0, len(b))
	for _, bk := range b {
		for _, eq := range bk.Activity.Equipment {
			if e.ID == eq.ItemID {
				bm = append(bm, bk)
			}
		}
	}

	return bm, nil
}

// Usage returns the number of these items which are requisitioned for use
// between the given time periods.
func (e *EquipmentItem) Usage(start, end time.Time) (int, error) {
	bk, err := e.Bookings(start, end)
	if err != nil {
		return 0, fmt.Errorf("%s usage: %w", e.Name, err)
	}

	tot := 0
	for _, b := range bk {
		for _, eq := range b.Activity.Equipment {
			if e.ID == eq.ID {
				tot += int(eq.Quantity)
			}
		}
	}

	return tot, nil
}

// DailyUsage returns the usage of the item on the given day. The begin and end
// period are taken as midnight on the given day up to midnight on the next
// day.
func (e *EquipmentItem) DailyUsage(t time.Time) (int, error) {
	start, end := t, t.Add(24*time.Hour)
	return e.Usage(start, end)
}

// NetQuantity returns the net balance for this item between the given times.
// If more items are requisitioned than are available, this value is negative.
func (e *EquipmentItem) NetQuantity(start, end time.Time) (int, error) {
	u, err := e.Usage(start, end)
	return int(e.Quantity) - u, err
}

// UseDB updates the internal database to a new instance. This shouldn't really
// be used unless really needed.
func (e *EquipmentItem) UseDB(db *gorm.DB) {
	e.db = db
}

// GetEquipmentItems returns all equipment stored in the inventory table.
func GetEquipment(db *gorm.DB) ([]EquipmentItem, error) {
	var eq []EquipmentItem
	res := db.Find(&eq)
	if res.Error != nil {
		return nil, fmt.Errorf("get equipment: %w", res.Error)
	}

	for i := range eq {
		eq[i].db = db
	}

	return eq, nil
}
