package data

import (
	"database/sql/driver"
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// Password minimum required security constants.
const (
	// Minimum password length in UTF-8 runes.
	MinPassLength = 5
	// Minimum password hash generation cost.
	PassHashCost = bcrypt.DefaultCost
)

// User password errors. These are intentionally capitalized, as they will
// likely be used in the UI.
var (
	ErrPassLength      = errors.New("Password must be at least six characters")
	ErrPassComplexity  = errors.New("Password must contain a lower-case and upper-case letter, as well as a number")
	ErrPassUsername    = errors.New("Password cannot be the user's name or email")
	ErrPassBlacklisted = errors.New("Password is blacklisted as too common")
)

// containsUpper returns true if an upper case character is present in set.
func containsUpper(set []rune) bool {
	for _, r := range set {
		if unicode.IsUpper(r) {
			return true
		}
	}

	return false
}

// containsUpper returns true if a lower case character is present in set.
func containsLower(set []rune) bool {
	for _, r := range set {
		if unicode.IsLower(r) {
			return true
		}
	}

	return false
}

// containsUpper returns true if a numeric character is present in set.
func containsNumber(set []rune) bool {
	for _, r := range set {
		if unicode.IsDigit(r) {
			return true
		}
	}

	return false
}

type Password struct {
	hashed []byte
}

// Matches returns true if pw and the hashed password stored by p are the same.
func (p *Password) Matches(pw string) bool {
	if p == nil {
		return false
	}

	return bcrypt.CompareHashAndPassword(p.hashed, []byte(pw)) == nil
}

// Value returns the SQLified value for the hashed password buffer.
func (p *Password) Value() (driver.Value, error) {
	if p == nil {
		return "", nil
	}

	return string(p.hashed), nil
}

// Scan reads the incoming password from the database, expecting that it is of
// type string, and stores it into this password.
func (p *Password) Scan(src interface{}) error {
	p.hashed = []byte(src.([]uint8))
	return nil
}

// Set checks that password meets all minimum security requirements before
// hashing the password and storing in the password buffer.
func (p *Password) Set(password string, u *User) error {
	dec := []rune(password)

	// Minimum length check.
	if len(dec) < MinPassLength {
		return ErrPassLength
	}
	// Minimum complexity check.
	if !containsLower(dec) || !containsUpper(dec) || !containsNumber(dec) {
		return ErrPassComplexity
	}
	// Duplication of username check.
	if password == u.Username || password == u.Email {
		return ErrPassUsername
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), PassHashCost)
	if err != nil {
		return err
	}

	p.hashed = hash
	return nil
}
