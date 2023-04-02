package errcat

import (
	"strings"
	"time"

	"github.com/agschwender/errcat-go/breaker"
	"github.com/agschwender/errcat-go/fallback"
	"github.com/agschwender/errcat-go/retrier"
	"github.com/agschwender/errcat-go/timer"
)

type call struct {
	args       map[string]interface{}
	categories []string
	duration   time.Duration
	err        error
	name       string
	startedAt  time.Time
}

type CallFn func() error

type Caller struct {
	dependency string
	key        string
	name       string

	breaker  *breaker.Breaker
	fallback *fallback.Fallback
	retrier  *retrier.Retrier
	timer    *timer.Timer
}

func New(dependency, name string) Caller {
	return Caller{
		dependency: dependency,
		name:       name,
		key:        strings.Join([]string{dependency, name}, ":"),
	}
}

// WithBreaker attaches a circuit breaker to the caller.
func (c Caller) WithBreaker(b *breaker.Breaker) Caller {
	c.breaker = b
	return c
}

// WithFallback defines the fallback behavior for the caller.
func (c Caller) WithFallback(f *fallback.Fallback) Caller {
	c.fallback = f
	return c
}

// WithRetrier indicates the caller should be retried in the event of a
// failure.
func (c Caller) WithRetrier(r *retrier.Retrier) Caller {
	c.retrier = r
	return c
}

// WithTimeout enforces a timeout on the caller. This method should only
// be used in those cases where the wrapped dependency does not already
// provide timeout functionality. This is because this method does not
// stop the callback from running if it exceeds the timeout; it only
// ensures that the Call returns in the allotted time, whereas the
// dependency functionality may provide better cleanup.
func (c Caller) WithTimeout(timeout time.Duration) Caller {
	c.timer = timer.New(timeout)
	return c
}

// Call executes the callback function.
func (c Caller) Call(cb CallFn) error {
	err := c.timer.Run(func() error {
		return c.breaker.Run(func() error {
			return c.retrier.Run(func() error {
				return cb()
			})
		})
	})

	if c.fallback.UseFallback(err) {
		return c.fallback.Call()
	}
	return err
}
