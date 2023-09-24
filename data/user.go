package data

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

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

// User lookup errors.
var (
	ErrInvalidID    = errors.New("invalid user ID")
	ErrInvalidName  = errors.New("invalid username")
	ErrUserNotFound = errors.New("user not found")
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

// GetUser selects the first user from the given database with the given user
// ID. Errors returned will either be due to a non-existent user, an SQL
// error or an invalid ID (== 0).
func GetUser(db *gorm.DB, id uint) (User, error) {
	if id == 0 {
		return User{}, fmt.Errorf("get user %d: %w", id, ErrInvalidID)
	}

	u := User{Model: &gorm.Model{ID: id}}
	if err := db.Where(&u).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return User{}, fmt.Errorf("get user %d: %w", id, ErrUserNotFound)
		}

		return User{}, fmt.Errorf("get user %d: sql error: %w", id, err)
	}

	return u, nil
}

// GetUserByName selects the first user from the given database with the given
// user name. Errors returned will either be due to a non-existent user, an SQL
// error or an invalid name (=="").
func GetUserByName(db *gorm.DB, name string) (User, error) {
	if name == "" {
		return User{}, fmt.Errorf("get user %s: %w", name, ErrInvalidName)
	}

	u := User{Username: name}
	if err := db.Where(&u).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return User{}, fmt.Errorf("get user %s: %w", name, ErrUserNotFound)
		}

		return User{}, fmt.Errorf("get user %s: sql error: %w", name, err)
	}

	return u, nil
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
