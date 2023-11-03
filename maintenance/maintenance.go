package maintenance

import (
	"errors"
	"log"
	"sync"
	"time"
)

// A Manager is a thread-safe structure which is able to be checked for
// maintenance mode.
type Manager struct {
	*sync.RWMutex
	is      bool
	entered time.Time

	// Read only after init
	log bool
}

// NewManager returns a new blank manager with a valid but unlocked mutex.
func NewManager(log bool) Manager {
	return Manager{RWMutex: new(sync.RWMutex), log: log}
}

// Is returns if the site is currently in maintenance mode.
func (m Manager) Is() bool {
	m.RLock()
	defer m.RUnlock()

	return m.is
}

// When returns the timestamp of when the most recent maintenance began. When
// panics if maintenance is not ongoing.
func (m Manager) When() time.Time {
	m.RLock()
	defer m.RUnlock()

	if !m.is {
		panic("use of maintenance.When while not in maintenance")
	}

	return m.entered
}

// State returns if maintenance is currently ongoing and the timestamp of when
// it began, if it is ongoing. If maintenance is not ongoing, the timestamp is
// nil. This is useful to avoid a race between checking lock state and
// obtaining entry timestamp.
func (m Manager) State() (bool, *time.Time) {
	m.RLock()
	defer m.RUnlock()

	if !m.is {
		return m.is, nil
	}

	// Copying out to avoid direct access to internals.
	tmp := m.entered
	return m.is, &tmp
}

// Enter waits for an exclusive lock on the manager before entering maintenance
// mode. This will cause all other clients to block before entering the mode.
// If, upon acquiring the lock, the manager is already in maintenance mode, an
// error is returned.
//
// Essentially, if this function returns with a nil error, a request may treat
// it as though it has an exclusive lock on the site.
func (m *Manager) Enter() error {
	m.Lock()
	defer m.Unlock()

	if m.is {
		return errors.New("already in maintenance mode")
	}

	if m.log {
		log.Println("[MAINTENANCE] Entering maintenance mode")
	}

	m.is = true
	m.entered = time.Now()
	return nil
}

// Exit waits for an exclusive lock on the manager before exiting maintenance.
// If the site is not currently in maintenance, Exit panics.
func (m *Manager) Exit() {
	m.Lock()
	defer m.Unlock()

	if !m.is {
		panic("request exited maintenance mode when not entered")
	}

	if m.log {
		log.Println("[MAINTENANCE] Exiting maintenance mode")
	}

	m.is = false
	m.entered = time.Time{}
}
