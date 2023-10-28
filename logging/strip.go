package logging

import (
	"regexp"
	"unicode"
)

var escapeMatcher = regexp.MustCompile("\x1b\\[[0-9;]*[mGKHF]")

// StripColors strips all ANSI coloring information from the datastream. This
// is not an inexpensive call, so please use sparingly. The returned stream is
// guaranteed to be valid UTF-8.
func StripColors(in []byte) []byte {
	w := escapeMatcher.ReplaceAll(in, []byte(""))

	dec := []rune(string(w))
	work := make([]rune, 0, len(dec))

	for _, r := range dec {
		if unicode.IsPrint(r) || r == '\n' || r == '\t' {
			work = append(work, r)
		}
	}

	return []byte(string(work))
}
