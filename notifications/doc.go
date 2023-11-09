// Package notifications implements a data structure for the temporary storage
// and delivery of notifications to users.
//
// Notifications are implemented via a FIFO queue for each user. The queue is
// created on demand when a new notification is popped. This is implemented via
// a map between the user and his notification queue.
//
// The notification FIFO queue only implements the most basic operations
// deliberately in order to minimize the possibility of strange things
// happening to users' notifications.
package notifications
