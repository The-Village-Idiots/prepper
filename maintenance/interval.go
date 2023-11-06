package maintenance

import (
	"context"
	"time"
)

// An Interval is a representation of a specific time interval between
// automated maintenance periods. It provides a channel which sends a
// notification message when it is the correct time for maintenance to be
// performed. Other functions are for managing the lifetime of the routine,
// such as starting and stopping.
type Interval interface {
	// Start initializes the producer goroutine for this interval and
	// begins any associated timers. This must not be called more than once
	// per object; doing so results in a race and a probable crash.
	Start(parent context.Context)
	// Stop shuts down the producer goroutine for this interval.
	Stop()
	// Chan returns the notification channel which will fire when the
	// interval has elapsed. The returned channel shall never be closed;
	// once no more events may come, the channel shall block forever.
	Chan() chan struct{}
}

// A stoppable is an object which contains a goroutine which may start or stop
// at will. This is used to implement consistent functionality in multiple
// interval runners.
type stoppable struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// Start initializes the contexts used for this cancellable object.
func (s *stoppable) Start(parent context.Context) {
	c := parent
	if c == nil {
		c = context.Background()
	}

	s.ctx, s.cancel = context.WithCancel(c)
}

// Stop cancels the context associated with this stoppable object.
func (s *stoppable) Stop() {
	s.cancel()
}

// a channer is a type which is capable of returning a channel of an empty
// struct.
type channer struct {
	c chan struct{}
}

func (c channer) Chan() chan struct{} {
	return c.c
}

// timeResolution returns the increment duration of the maximum resolution
// possible from the given time.
func timeResolution(t time.Time) time.Duration {
	switch {
	case t.Second() != 0:
		return time.Second
	case t.Minute() != 0:
		return time.Minute
	case t.Hour() != 0:
		fallthrough
	default:
		return time.Hour
	}
}

// Daily is an interval which fires daily at the same instant in the day. This
// is performed by checking the current time at the minimum nonzero time
// denominator, at a maximum resolution of seconds. The date component of the
// supplied time is assumed to be zero but is otherwise ignored.
type Daily struct {
	*stoppable
	channer
	Time time.Time

	int time.Duration
}

// compareTime compares the current time to that on file. This is needed to
// deliberately discount date information.
func (d *Daily) compareTime(t time.Time) bool {
	w := d.Time.Truncate(d.int)

	if t.Hour() == w.Hour() &&
		t.Minute() == w.Minute() &&
		t.Second() == w.Second() {

		return true
	}

	return false
}

func (d *Daily) run() {
	tk := time.NewTicker(d.int)
	defer tk.Stop()

	for {
		select {
		case <-tk.C:
			if d.compareTime(time.Now().Truncate(d.int)) {
				d.c <- struct{}{}
			}
		case <-d.ctx.Done():
			return
		}
	}
}

func (d *Daily) Start(parent context.Context) {
	d.stoppable = &stoppable{}
	d.stoppable.Start(parent)

	d.int = timeResolution(d.Time.Truncate(time.Second))
	d.c = make(chan struct{})

	go d.run()
}

// Regularly is an interval which fires regularly after regular intervals. This
// is a much simpler interface than that used by Daily as we do not need to
// account for comparing dates and times.
type Regularly struct {
	*stoppable
	channer
	Interval time.Duration
}

func (r *Regularly) run() {
	tk := time.NewTicker(r.Interval)
	defer tk.Stop()

	for {
		select {
		case <-tk.C:
			r.c <- struct{}{}
		case <-r.ctx.Done():
			return
		}
	}
}

func (r *Regularly) Start(parent context.Context) {
	r.stoppable = &stoppable{}
	r.stoppable.Start(parent)

	r.c = make(chan struct{})
	go r.run()
}
