package isams

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// ID is a convenience parser for IDs, automatically converting between strings
// and ints.
type ID uint64

func (i *ID) UnmarshalJSON(date []byte) error {
	var v string
	err := json.Unmarshal(date, &v)
	if err != nil {
		return fmt.Errorf("unmarshal isams id: %w", err)
	}

	uid, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return fmt.Errorf("unmarshal isams id: %w", err)
	}

	*i = ID(uid)
	return nil
}
