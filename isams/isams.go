package isams

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	ErrRequestIO   = errors.New("request I/O failed")
	ErrEncoding    = errors.New("bad response encoding")
	ErrPrepopulate = errors.New("prepopulation failed")

	ErrNotFound = errors.New("not found")
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

	Users []User
}

func (i *ISAMS) request() (*isamsResponse, error) {
	if i.client == nil {
		return nil, ErrInit
	}

	resp, err := i.client.Get(i.endpoint.String())
	if err != nil {
		return nil, fmt.Errorf("isams at %s: %w: %s", i.endpoint.String(), ErrRequest, err.Error())
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("isams at %s: %w: response code %d", i.endpoint.String(), ErrRequest, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("isams at %s: %w: %s", i.endpoint.String(), ErrRequestIO, err.Error())
	}
	defer resp.Body.Close()

	// NOTE: The structure of this struct is really important. DO NOT CHANGE!
	obj := struct {
		I isamsResponse `json:"iSAMS"`
	}{}

	err = json.Unmarshal(body, &obj)
	if err != nil {
		return nil, fmt.Errorf("isams at %s: %w: %s", i.endpoint.String(), ErrEncoding, err.Error())
	}

	return &obj.I, nil
}

// prepopulate loads startup data from the database to prepopulate exported
// fields.
func (i *ISAMS) prepopulate() error {
	resp, err := i.request()
	if err != nil {
		return err
	}

	i.Users = resp.HRManager.CurrentStaff.StaffMember

	return nil
}

// FindUser looks up a user by usercode, which should be the ID used across the
// rest of the codebase to uniquely identify a user (due to it being used
// across the API).
func (i *ISAMS) FindUser(ucode string) (*User, error) {
	for _, u := range i.Users {
		if u.UserCode == ucode {
			return &u, nil
		}

	}

	return nil, fmt.Errorf("find isams user %s: %w", ucode, ErrNotFound)
}

// New loads a new intance of the ISAMS data manager. The first action is to
// reach out to the ISAMS database to retrieve basic information about
// timetables etc. This also verifies that the connection has been set up
// correctly.
func New(domain, key string) (*ISAMS, error) {
	obj := &ISAMS{
		isamsEndpoint{domain, key},
		&http.Client{},

		[]User{},
	}

	if err := obj.prepopulate(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrPrepopulate, err)
	}

	return obj, nil
}
