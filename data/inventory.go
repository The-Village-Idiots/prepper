package data

import "gorm.io/gorm"

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
}
