package isams

import (
	"errors"
	"fmt"
	"net/http"
)

// ISAMS request detail constants.
const (
	apiEndpoint = "/api/batch/1.0/json.ashx?apiKey="
)

// ISAMS usage errors.
var (
	ErrInit        = errors.New("not initialized")
	ErrRequest     = errors.New("request failed")
	ErrPrepopulate = errors.New("prepopulation failed")
)

// isamsEndpoint is a configured requestable endpoint, formatted to request the
// correct API via HTTPS.
type isamsEndpoint struct {
	domain, key string
}

func (i isamsEndpoint) String() string {
	return fmt.Sprintf("https://%s%s{%s}", i.domain, apiEndpoint, i.key)
}

// ISAMS is the data manager for a live ISAMS connection. Any exported fields
// are pre-populated at startup and are not written to again (and as such
// should not be modified).
type ISAMS struct {
	endpoint isamsEndpoint
	client   *http.Client
}

func (i *ISAMS) request() (*http.Response, error) {
	if i.client == nil {
		return nil, ErrInit
	}

	resp, err := i.client.Get(i.endpoint.String())
	if err != nil {
		return resp, fmt.Errorf("isams at %s: %w: %s", i.endpoint.String(), ErrRequest, err.Error())
	}

	return resp, nil
}

// prepopulate loads startup data from the database to prepopulate exported
// fields.
func (i *ISAMS) prepopulate() error {
	return ErrPrepopulate
}

// NewISAMS loads a new intance of the ISAMS data manager. The first action is
// to reach out to the ISAMS database to retrieve basic information about
// timetables etc. This also verifies that the connection has been set up
// correctly.
func NewISAMS(domain, key string) (*ISAMS, error) {
	obj := &ISAMS{
		isamsEndpoint{domain, key},

		&http.Client{},
	}

	if err := obj.prepopulate(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrPrepopulate, err)
	}

	return obj, nil
}
