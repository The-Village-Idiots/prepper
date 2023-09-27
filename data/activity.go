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

	// If an activity is temporary, it can be deleted after the booking has
	// passed.
	Temporary bool

	// Used to link to individual EquipmentItem(s).
	// Link via foreign key in EquipmentSet.
	Equipment []EquipmentSet
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
