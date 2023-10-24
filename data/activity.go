package data

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// Activity retrieval errors.
var (
	ErrInvalidActivityID = errors.New("invalid activity ID")
	ErrActivityNotFound  = errors.New("activity not found")
)

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

// Temp returns this activity modified to be suitable for use as a temporary
// activity.
func (a Activity) Temp() Activity {
	act := a

	// Overwrite with generic values
	act.Title = "Temporary Activity"
	act.Description = fmt.Sprintf("Temporary activity copied from \"%s\"", a.Title)
	act.Temporary = true

	// Blank data which we expect to fill in later
	act.Model = nil
	act.OwnerID = 0
	act.Owner = User{}

	// Update equipment set
	for i := range act.Equipment {
		act.Equipment[i].Model = nil
		act.Equipment[i].ActivityID = 0
	}

	return act
}

// Clone creates a new temporary activity cloned from this one, which will have
// a unique ID and an overwritten model for each contained primary key. All
// foreign keys will also be cloned and marked as temporary. Extras will be
// appended to the equipment set.
func (a Activity) Clone(db *gorm.DB, owner uint, extras []EquipmentSet) (Activity, error) {
	act := a.Temp()

	act.Equipment = append(a.Equipment, extras...)
	act.OwnerID = owner

	err := db.Create(&act).Error
	if err != nil {
		return Activity{}, fmt.Errorf("clone activity: sql error: %w", err)
	}

	return act, nil
}

// GetActivity retrieves an activity from the database by ID, with all foreign
// keys joined.
func GetActivity(db *gorm.DB, id uint) (Activity, error) {
	if id == 0 {
		return Activity{}, fmt.Errorf("get activity %d: %w", id, ErrInvalidActivityID)
	}

	a := Activity{Model: &gorm.Model{ID: id}}
	res := db.Where(&a).
		Joins("Owner").
		Preload("Equipment").Preload("Equipment.Item").
		First(&a)

	if err := res.Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Activity{}, fmt.Errorf("get activity %d: %w", id, ErrActivityNotFound)
		}

		return Activity{}, fmt.Errorf("get activity %d: sql error: %w", id, err)
	}

	return a, nil
}

// GetActivities returns all activities stored in the database with all foreign
// keys joined.
func GetActivities(db *gorm.DB) ([]Activity, error) {
	var acts []Activity
	res := db.Find(&acts).
		Joins("Owner").
		Preload("Equipment").Preload("Equipment.Item")

	if res.Error != nil {
		return nil, fmt.Errorf("get activities: %w", res.Error)
	}

	return acts, nil
}

// GetPermanentActivities returns all activities stored in the database which
// are not marked as temporary with all foreign keys filled.
func GetPermanentActivities(db *gorm.DB) ([]Activity, error) {
	var acts []Activity
	res := db.Find(&acts).
		Where(&Activity{Temporary: false}).
		Joins("Owner").
		Preload("Equipment").Preload("Equipment.Item")

	if res.Error != nil {
		return nil, fmt.Errorf("get activities: %w", res.Error)
	}

	return acts, nil
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
	ActivityID uint `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	// Quantity requisitioned for this activity.
	Quantity uint
	// Marked as important if vital for the activity to succeed. Use
	// responsibly!
	Important bool

	ItemID uint
	Item   EquipmentItem `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
