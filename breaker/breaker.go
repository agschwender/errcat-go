package breaker

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrBreakerOpen indicates that the breaker is in the open state and
// will not run the supplied callback. This error will also be returned
// in the half open state and at capacity.
var ErrBreakerOpen = errors.New("circuit breaker is open")

type Status uint8

func (s Status) String() string {
	switch s {
	case Closed:
		return "closed"
	case HalfOpen:
		return "half-open"
	case Open:
		return "open"
	default:
		return "unknown"
	}
}

const (
	// Closed indicates the callback function will be run as normal.
	Closed Status = 0

	// HalfOpen indicates the callback function will be run in a limited
	// capacity and a failure will return it to the open state.
	HalfOpen Status = 1

	// Open indicates the callback function will not be run while in
	// this state.
	Open Status = 2

	defaultMaxFailures = uint(5)
	defaultMaxRequests = uint(1)
	defaultTimeout     = time.Duration(60) * time.Second
)

var defaultIsFailure = func(err error) bool { return err != nil }

// State maintains the state of the circuit breaker, which includes the
// status and relevant counts.
type State struct {
	status    Status
	expiresAt time.Time

	// The number of consecutive failures. This is only tracked when the
	// circuit breaker is in the closed state.
	Failures uint

	// The number of consecutive successes. This is only tracked when
	// the circuit breaker is in the half open state.
	Successes uint

	now func() time.Time
}

// Status returns circuit breaker status.
func (s State) Status() Status {
	if s.status == Open && !s.expiresAt.After(s.now()) {
		return HalfOpen
	}
	return s.status
}

func (s *State) failure() {
	s.Failures++
	s.Successes = 0
}

func (s *State) success() {
	s.Successes++
	s.Failures = 0
}

// Breaker provides the logic for conditionally calling a function based
// on its passed performance.
type Breaker struct {
	isFailure   func(err error) bool
	maxFailures uint
	maxRequests uint
	now         func() time.Time
	timeout     time.Duration

	lock     sync.RWMutex
	requests uint
	state    State
}

type option func(*Breaker)

// New creates a new Breaker with the supplied options.
func New(opts ...option) *Breaker {
	b := &Breaker{
		isFailure:   defaultIsFailure,
		maxFailures: defaultMaxFailures,
		maxRequests: defaultMaxRequests,
		now:         time.Now,
		timeout:     defaultTimeout,
	}

	for _, opt := range opts {
		opt(b)
	}

	b.state = State{now: b.now}

	return b
}

// WithIsFailure defines the logic for determining if the error should be
// counted toward the breaker failures.
func WithIsFailure(isFailure func(err error) bool) option {
	return func(b *Breaker) {
		if isFailure == nil {
			isFailure = defaultIsFailure
		}
		b.isFailure = isFailure
	}
}

// WithMaxRequests set the maximum number of requests that should be
// made while in the half open state.
func WithMaxHalfOpenRequests(maxRequests uint) option {
	return func(b *Breaker) {
		if maxRequests == 0 {
			maxRequests = defaultMaxRequests
		}
		b.maxRequests = maxRequests
	}
}

// WithMaxFailures indicates the breaker should go into the open
// state after reaching the supplied number of consecutive failures.
func WithMaxFailures(maxFailures uint) option {
	return func(b *Breaker) {
		if maxFailures == 0 {
			maxFailures = defaultMaxFailures
		}
		b.maxFailures = maxFailures
	}
}

// WithNow sets the function for getting the current time. This is only
// useful for testing.
func WithNow(now func() time.Time) option {
	return func(b *Breaker) {
		if now == nil {
			now = time.Now
		}
		b.now = now
	}
}

// WithTimeout sets the duration over which the breaker will remain in
// the open state.
func WithTimeout(timeout time.Duration) option {
	return func(b *Breaker) {
		if timeout == 0 {
			timeout = defaultTimeout
		}
		b.timeout = timeout
	}
}

// Run executes the callback if the circuit breaker is not in the open
// state. It will track successes and failures in order to determine the
// state.
func (b *Breaker) Run(cb func() error) (err error) {
	if b == nil {
		return cb()
	}

	state := b.State()
	status := state.Status()
	if status == Open {
		return ErrBreakerOpen
	}
	if status == HalfOpen && !b.canMakeHalfOpenRequest() {
		return ErrBreakerOpen
	}

	err = b.safeRun(cb)
	b.handleError(state, err)

	return err
}

// State returns the current breaker state.
func (b *Breaker) State() State {
	if b == nil {
		return State{now: time.Now}
	}

	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.state
}

func (b *Breaker) canMakeHalfOpenRequest() bool {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.requests >= b.maxRequests {
		return false
	}

	b.requests++
	return true
}

func (b *Breaker) handleError(state State, err error) {
	isFailure := err != nil && b.isFailure(err)

	// Since we only track failures in the closed state, we can exit
	// early without accessing the lock as long as we do not need reset
	// the failures.
	if state.status == Closed && !isFailure && state.Failures == 0 {
		return
	}

	b.lock.Lock()
	defer b.lock.Unlock()

	switch b.state.Status() {
	case Closed:
		// When in the closed state, we track failures only and check if
		// the circuit breaker should transition into the open state.
		if isFailure {
			b.state.failure()
			if b.shouldOpen() {
				b.setState(Open)
			}
		} else {
			b.state.Failures = 0
		}
	case HalfOpen:
		// When in the half-open state, a failure will return the
		// circuit breaker to the open state. In order to transition to
		// the closed state, it must receive a success for each of its
		// allowed half-open requests.
		if isFailure {
			b.setState(Open)
		} else {
			b.state.success()
			if b.state.Successes == b.maxRequests {
				b.setState(Closed)
			}
		}
	}

}

func (b *Breaker) safeRun(cb func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	err = cb()
	return
}

func (b *Breaker) setState(status Status) {
	// Assumes that a lock has already been taken for writing to the
	// state and counts variables.
	b.requests = 0
	b.state = State{status: status, now: b.now}
	if status == Open {
		b.state.expiresAt = b.now().Add(b.timeout)
	}
}

func (b *Breaker) shouldOpen() bool {
	return b.state.Failures >= b.maxFailures
}
