package timer

import (
	"errors"
	"fmt"
	"time"
)

// ErrTimeout indicates that the function run by the timer failed to
// complete before the timer ran out.
var ErrTimeout = errors.New("timeout exceeded")

type Timer struct {
	duration time.Duration
}

// New creates a new Timer with the supplied duration.
func New(d time.Duration) *Timer {
	return &Timer{duration: d}
}

// Run executes the callback ensuring it returns by the timeout
// duration. Note, this does NOT kill the function call, it will still
// complete in the background. In general, it is preferable timeout
// functionality provided by the client of the dependency. However in
// those cases where that functionality is not provided, this timer
// functionality may be appropriate.
func (t *Timer) Run(cb func() error) error {
	if t == nil || t.duration <= time.Duration(0) {
		return cb()
	}

	timer := time.NewTimer(t.duration)
	defer timer.Stop()

	done := make(chan error)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("%v", r)
			}
		}()

		done <- cb()
	}()

	select {
	case <-timer.C:
		return ErrTimeout
	case err := <-done:
		return err
	}
}
