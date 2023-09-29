package isams

import (
	"encoding/json"
	"fmt"
)

// Bool is a decoder for iSAMS boolean values, which are stored as strings (for
// some reason).
type Bool bool

// UnmarshalJSON automatically detects if the string value being unmarshalled
// is truthy and stores into the isams.Bool object.
func (b *Bool) UnmarshalJSON(data []byte) error {
	var v string
	err := json.Unmarshal(data, &v)
	if err != nil {
		return fmt.Errorf("unmarshal isams bool: %w", err)
	}

	if v == "1" || v == "true" {
		*b = true
	}

	*b = false
	return nil
}
