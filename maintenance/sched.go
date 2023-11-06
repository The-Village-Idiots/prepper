package maintenance

import (
	"context"
	"fmt"
	"strings"
)

// Error is an aggregated list of errors which occurred during routine
// maintenance. In the stringified output, a summary line is printed, followed
// by each component error on a separate newline proceeded by a tab and a dash
// to form a list.
type Error []error

func (m Error) Error() string {
	sb := &strings.Builder{}

	fmt.Fprintln(sb, len(m), "errors performing maintenance:")
	for _, e := range m {
		fmt.Fprintln(sb, "\t-", e.Error())
	}

	return sb.String()
}

// A Scheduler is responsible for scheduling routine maintenance, enabling
// maintenance mode on the server before performing any tasks registered for
// maintenance. Handling of timings is performed by any object which satisfies
// the Interval interface.
type Scheduler struct {
	// Interval is the interval between maintenance periods. Please see the
	// documentation for those types.
	Interval Interval
	// Manager is the maintenance manager which this scheduler will
	// schedule maintenance for.
	Manager *Manager
	// Handlers are the functions to be called during maintenance time.
	Handlers []func() error
	// Ctx is the context for this scheduler. When cancelled, the worker
	// goroutine dies.
	Ctx context.Context
	// Err is the channel on which errors are sent. if Err is nil, errors
	// are silently discarded.
	Err chan error
}

func (s Scheduler) do() {
	s.Manager.Enter()
	defer s.Manager.Exit()

	e := make(Error, 0, len(s.Handlers))
	for _, fn := range s.Handlers {
		err := fn()
		if err != nil {
			e = append(e, err)
		}
	}

	if s.Err != nil && len(s.Err) > 0 {
		s.Err <- e
	}
}

// Run blocks the calling goroutine until Scheduler shuts down, calling all
// handlers each time Interval elapses.
func (s Scheduler) Run() {
	s.Interval.Start(s.Ctx)
	defer s.Interval.Stop()

	for {
		select {
		case <-s.Interval.Chan():
			s.do()
		case <-s.Ctx.Done():
			return
		}
	}
}
