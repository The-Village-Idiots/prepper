package data

import "strings"

// CumulativeError is a type which allows various error values to collect and
// accumulate into a slice which formats them nicely. For nice use by
// consumers, please use the provided Return method to get the value which
// should be returned to the consumer. This is to allow the idiom of err != nil
// to still work correctly (as, for example, returning an empty CumulativeError
// as an error interface *does not* satisfy this regular check!).
type CumulativeError []error

// Error formats all the contained error messages into a nice, multi-line error
// message. Each separate error is prepended with a dash for a bullet point and
// a tab character for indentation.
func (c CumulativeError) Error() string {
	sb := strings.Builder{}

	for _, e := range c {
		sb.WriteString("\t- " + e.Error() + "\n")
	}

	return sb.String()
}

// Returns returns the value which should be returned to the caller upon use.
// This is to allow the idiomatic error nil check to be properly satisfied.
func (c CumulativeError) Return() error {
	if len(c) == 0 {
		return nil
	}

	return c
}

// Push adds a new error onto the back of the cumulative error stack.
func (c *CumulativeError) Push(e error) {
	*c = append(*c, e)
}
