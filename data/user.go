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

const DefaultPassword = "DefaultPassword1234"

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

	Username     string    `json:"username"`
	Password     *Password `json:"-"`
	PasswordHint string    `json:"password_hint"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	Title        string    `json:"title"`
	Role         UserRole  `json:"role"`
	Email        string    `json:"email"`
	Telephone    string    `json:"telephone"`
}

// NewUser generates a new dummy user of the specified role, returning a user
// object with a valid username and ID such that it can be later updated. The
// default password is set to a sensible default. All other fields are left
// unset.
func NewUser(db *gorm.DB, role UserRole) (User, error) {
	var u User
	for newi := 1; newi < 10; newi++ {
		name := fmt.Sprint("newuser", newi)
		u = User{
			Username:     name,
			Password:     &Password{},
			PasswordHint: "Default Password",
			Role:         role,
		}

		u.SetPassword(DefaultPassword)

		// Break when username not found
		if db.Model(&u).Where("username = ?", name).First(&u).Error != nil {
			break
		}
	}

	if err := db.Create(&u).Error; err != nil {
		return u, fmt.Errorf("create temp user %s: sql error: %w", u.Username, err)
	}

	return u, nil
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

// GetUsers returns all users stored in the database.
func GetUsers(db *gorm.DB) ([]User, error) {
	var us []User
	res := db.Find(&us)
	if res.Error != nil {
		return nil, fmt.Errorf("get users: %w", res.Error)
	}

	return us, nil
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

// Can returns true if the given user is capable of performing the given action
// act.
func (u User) Can(act uint8) bool {
	return u.Role >= UserRole(act)
}

func (u User) IsTechnician() bool {
	return u.Role >= UserTechnician
}

func (u User) IsAdmin() bool {
	return u.Role >= UserAdmin
}

// DisplayName returns the name which we should prefer to display on the user's
// end. This is not machine-friendly.
func (u User) DisplayName() string {
	// First of all, try the first name
	if u.FirstName != "" {
		return u.FirstName
	}

	// Next try last
	if u.LastName != "" {
		// Add title for politeness!
		if u.Title != "" {
			return fmt.Sprintf("%s. %s", u.Title, u.LastName)
		}

		return u.LastName
	}

	// Finally resort to username
	return u.Username
}
