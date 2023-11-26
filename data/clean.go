package data

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// cleanTable cleans the table for a specific model.
func cleanTable(db *gorm.DB, model any) (int64, error) {
	res := db.Unscoped().Model(model).Where("deleted_at < ?", time.Now().Add(-3*7*24*time.Hour)).Delete(&model)
	if err := res.Error; err != nil {
		return res.RowsAffected, fmt.Errorf("cleaning %T: sql error: %w", model, err)
	}

	// Reset auto-increment if possible
	stmt := gorm.Statement{DB: db}
	stmt.Parse(&model)
	ores := db.Exec(fmt.Sprintln("ALTER TABLE", stmt.Schema.Table, "AUTO_INCREMENT = 1"))
	if err := ores.Error; err != nil {
		return res.RowsAffected, fmt.Errorf("cleaning %T: reset autonumber: sql error: %w", model, err)
	}

	return res.RowsAffected, nil
}

// CleanDeleted walks over all tables in the database and cleans out things
// which were deleted over a week ago. If any of the operations fail, the
// entire operation is rolled back. This should probably only be run inside of
// maintenance!
func CleanDeleted(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// Note: Order here is important to avoid foreign key violations!
		cleanTable(tx, User{})
		cleanTable(tx, EquipmentSet{})
		cleanTable(tx, Activity{})
		cleanTable(tx, Booking{})
		cleanTable(tx, EquipmentItem{})

		return tx.Error
	})
}
