package notifications

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Notification types. Provide a visual distinction on the client.
const (
	// Generic. Normal white background.
	TypeGeneric = Type(iota)
	// Important. Primary background.
	TypeImportant
	// Danger. Bright red background. Usually for rejected prep requests.
	TypeDanger
	// Success. Bright green background.
	TypeSuccess
)

// maxQueueLength is the length at which the queue notification buffer will
// begin rejecting new entries.
const maxQueueLength = 15

var (
	ErrNoSuchUser = errors.New("no such user")
	ErrEmptyQueue = errors.New("queue empty")
)

// Type represents a type of notification, which is an enumerator over the
// constants defined above. Please see individual constants' definitions for
// more details.
type Type uint

// A Store contains a map between user IDs and their associated notification
// queues. All operations on a store are thread safe by design.
type Store struct {
	sync.Mutex
	store map[uint]notificationQueue
}

// PopUser pops a notification off the front of the given user's queue. If the
// user has not received any notifications in the past ErrNoSuchUser is
// returned. If the user has no notifications (the queue is empty but was not
// always), ErrEmptyQueue is returned.
func (s *Store) PopUser(user uint) (Notification, error) {
	s.Lock()
	defer s.Unlock()

	u, ok := s.store[user]
	if !ok {
		return Notification{}, fmt.Errorf("pop user %v: %w", user, ErrNoSuchUser)
	}
	defer func() { s.store[user] = u }()

	n, err := u.Pop()
	if err != nil {
		return Notification{}, fmt.Errorf("pop user %v: %w", user, err)
	}

	return n, nil
}

// PushUser pushes the given notification into the given user's notification
// queue. If the user has no queue yet, one will be fabricated. The new length
// of the user's queue is returned.
func (s *Store) PushUser(user uint, not Notification) int {
	s.Lock()
	defer s.Unlock()

	q, ok := s.store[user]
	if !ok {
		q = newNotificationQueue()
	}
	defer func() { s.store[user] = q }()

	return q.Push(not)
}

// NewStore allocates and returns a new notification store.
func NewStore() *Store {
	return &Store{
		sync.Mutex{},
		make(map[uint]notificationQueue),
	}
}

// A Notification represents a single entry in the notification queue.
type Notification struct {
	// Main notification content.
	Title string `json:"title"`
	Body  string `json:"body"`
	// Action link.
	Action string    `json:"action"`
	Time   time.Time `json:"time"`
	Type   Type      `json:"type"`
}

type notificationQueue struct {
	*sync.Mutex
	// Queue. Slice properties are implicitly used as r/w head.
	q []Notification
}

func newNotificationQueue() notificationQueue {
	return notificationQueue{
		new(sync.Mutex),
		make([]Notification, 0, maxQueueLength),
	}
}

// Clear truncates the queue to zero entries, resetting the read head in the
// process.
func (n *notificationQueue) Clear() {
	n.Lock()
	defer n.Unlock()

	n.clear()
}

// clear is Clear but assumes that a lock is held already.
func (n *notificationQueue) clear() {
	n.q = make([]Notification, 0, maxQueueLength)
}

// Push adds a new notification to the end of the queue, returning the new
// length.
func (n *notificationQueue) Push(not Notification) int {
	n.Lock()
	defer n.Unlock()

	if len(n.q) >= maxQueueLength {
		return len(n.q)
	}

	n.q = append(n.q, not)
	return len(n.q)
}

// Pop removes the leading notification from the queue. If this pop has removed
// all remaining notifications, the queue will be reset to avoid fragmentation.
func (n *notificationQueue) Pop() (Notification, error) {
	n.Lock()
	defer n.Unlock()

	if len(n.q) == 0 {
		return Notification{}, ErrEmptyQueue
	}

	ret := n.q[0]
	n.q = n.q[1:]
	if len(n.q) == 0 {
		n.clear()
	}

	return ret, nil
}

// Len returns the number of elements waiting to be popped.
func (n *notificationQueue) Len() int {
	n.Lock()
	defer n.Unlock()

	return len(n.q)
}
