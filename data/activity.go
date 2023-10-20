package data

import "gorm.io/gorm"

// An Activity is the details for a booking. It contains needed equipment and
// the quantities in which they are needed. It links (through the link table
// EquipmentSets) to an array of equipment which determines the quantity
// required etc.
type Activity struct {
	*gorm.Model

	Title       string
	Description string

	// To get categories, use SELECT DISTINCT.
	Category string

	// Determines who owns and can edit the activity.
	OwnerID uint
	Owner   User

	// Temporary activies are those which have an associated booking. All
	// others are to be used as templates for others and are copied upon
	// use.
	Temporary bool

	// Used to link to individual EquipmentItem(s).
	// Link via foreign key in EquipmentSet.
	Equipment []EquipmentSet
}

// ItemQuantity returns the number of the given item requisitioned for this
// activity, or zero if this item is not in use by this activity.
func (a Activity) ItemQuantity(i EquipmentItem) uint {
	for _, e := range a.Equipment {
		if e.ItemID == i.ID {
			return e.Quantity
		}
	}

	return 0
}

// EquipmentSet is the link table for equipment used in an activity.
type EquipmentSet struct {
	*gorm.Model
	ActivityID uint

	// Quantity requisitioned for this activity.
	Quantity uint
	// Marked as important if vital for the activity to succeed. Use
	// responsibly!
	Important bool

	ItemID uint
	Item   EquipmentItem
}
