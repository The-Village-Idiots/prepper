package data

// User capabilities. A capability is the integer value of the minimum role
// required to perform an action on the site. This means that anybody can
// perform an action that a teacher account can, but a teacher may not perform
// an action that a technician can.
const (
	// Do nothing. Anybody can do nothing.
	CapNull = UserTeacher

	// Add, delete or modify users.
	CapManageUsers = UserAdmin
	// Manage another user's timetable.
	CapManageTimetable = UserTechnician
	// Change passwords without authentication.
	CapResetPassword = UserAdmin
	// Become another user without login credentials.
	CapImpersonate = UserAdmin

	// Teachers can manage their own bookings.
	CapOwnBooking = UserTeacher
	// Only technicians may modify others' bookings.
	CapAllBooking = UserTechnician

	// Technicians may manage the inventory database.
	CapManageInventory      = UserTechnician
	CapManageOtherInventory = UserAdmin

	// View server logs.
	CapLogging = UserAdmin
)
