package data

import "gorm.io/gorm"

// User role definitions. Users with higher values for roles have more privileges.
const (
	// A teacher has the least privileges and can only book requisitions
	// and view their own timetable.
	UserTeacher = iota
	// A technician has more privileges than a teacher, being able to view
	// their own and others timetables, as well as managing the inventory
	// and much more.
	UserTechnician
	// An admin has every privilege the site has to offer.
	UserAdmin
)

// A UserRole is the enumerator type for each possible user role.
type UserRole int8

func (u UserRole) String() string {
	switch u {
	case UserTeacher:
		return "teacher"
	case UserTechnician:
		return "technician"
	case UserAdmin:
		return "admin"
	default:
		panic("use of invalid user role value")
	}
}

// A User is a user login record. It is uniquely identified by a user ID and is
// authenticated using a username and hashed password (along with a password
// hint just in case). All other fields are either cosmetic or for convenience.
type User struct {
	*gorm.Model

	Username     string
	Password     Password
	PasswordHint string
	FirstName    string
	LastName     string
	Title        string
	Role         UserRole
	Email        string
	Telephone    string
}

// Calls u.Password.Set with the current user as an argument.
func (u *User) SetPassword(pw string) error {
	return u.Password.Set(pw, u)
}

// Returns true if a user with the specified details exists.
// All provided fields are inspected for validity but null or zero value fields
// are ignored.
func (u *User) Exists(db *gorm.DB) bool {
	cpy := *u
	return db.Where(u).First(&cpy).Error == nil
}
