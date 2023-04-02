package breaker_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agschwender/errcat-go/breaker"
)

func TestAsNil(t *testing.T) {
	var b *breaker.Breaker
	assert.Equal(t, breaker.Closed, b.State().Status())

	counts := 0
	err := b.Run(func() error {
		counts++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, counts)
}

func TestStatusString(t *testing.T) {
	assert.Equal(t, "closed", breaker.Closed.String())
	assert.Equal(t, "half-open", breaker.HalfOpen.String())
	assert.Equal(t, "open", breaker.Open.String())
	assert.Equal(t, "unknown", breaker.Status(uint(100)).String())
}

func TestWithDefaults(t *testing.T) {
	now := time.Now()

	b := breaker.New(breaker.WithNow(func() time.Time { return now }))

	// Confirm happy path
	err := b.Run(func() error { return nil })
	require.NoError(t, err)

	// Not enough errors to trigger the open state
	for i := 0; i < 4; i++ {
		b.Run(func() error { return fmt.Errorf("oops") })
	}
	assert.Equal(t, breaker.Closed.String(), b.State().Status().String())
	assert.Equal(t, 4, int(b.State().Failures))

	// Reaches the max failures to trigger the open state
	err = b.Run(func() error { return fmt.Errorf("oops") })
	require.Error(t, err)
	assert.Equal(t, "oops", err.Error())
	assert.Equal(t, breaker.Open.String(), b.State().Status().String())

	// Confirm that it automatically returns an open error
	err = b.Run(func() error { return fmt.Errorf("oops") })
	require.Error(t, err)
	assert.Equal(t, breaker.ErrBreakerOpen, err)

	// Move the time to the timeout duration
	now = now.Add(time.Duration(60) * time.Second)
	assert.Equal(t, breaker.HalfOpen.String(), b.State().Status().String())

	// Force an error during half open state to trigger another timeout
	b.Run(func() error { return fmt.Errorf("oops") })
	assert.Equal(t, breaker.Open.String(), b.State().Status().String())

	// Move the time to the timeout duration again
	now = now.Add(time.Duration(60) * time.Second)
	assert.Equal(t, breaker.HalfOpen.String(), b.State().Status().String())

	// Send a slow success request
	first := make(chan bool)
	second := make(chan bool)
	done := make(chan bool)
	go func() {
		b.Run(func() error {
			second <- true
			<-first
			return nil
		})
		done <- true
	}()

	// Send another request that exceeds the max allowed requests during
	// the half open state.
	go func() {
		<-second
		err := b.Run(func() error { return nil })
		require.Error(t, err)
		assert.Equal(t, breaker.ErrBreakerOpen, err)
		first <- true
	}()

	<-done
	// The slow successful requests should land now and move the breaker
	// to the closed state.
	assert.Equal(t, breaker.Closed.String(), b.State().Status().String())
}

func TestWithNowNil(t *testing.T) {
	b := breaker.New(breaker.WithNow(nil))
	for i := 0; i < 10; i++ {
		b.Run(func() error { return fmt.Errorf("oops") })
	}
	assert.Equal(t, breaker.Open.String(), b.State().Status().String())
}

func TestWithOverrides(t *testing.T) {
	now := time.Now()

	b := breaker.New(
		breaker.WithNow(func() time.Time { return now }),
		breaker.WithIsFailure(func(err error) bool { return err != nil && err.Error() == "oops" }),
		breaker.WithMaxHalfOpenRequests(uint(2)),
		breaker.WithMaxFailures(uint(10)),
		breaker.WithTimeout(time.Duration(10)*time.Second),
	)
	for i := 0; i < 9; i++ {
		b.Run(func() error { return fmt.Errorf("oops") })
	}

	// Not enough errors to trigger the open state
	assert.Equal(t, breaker.Closed.String(), b.State().Status().String())
	assert.Equal(t, 9, int(b.State().Failures))

	// Send an error that is not considered a failure and verify that
	// consecutive failures is reset.
	b.Run(func() error { return fmt.Errorf("some other error") })
	assert.Equal(t, breaker.Closed.String(), b.State().Status().String())
	assert.Equal(t, 0, int(b.State().Failures))

	// Reaches the max failures to trigger the open state
	for i := 0; i < 9; i++ {
		b.Run(func() error { return fmt.Errorf("oops") })
	}
	err := b.Run(func() error { return fmt.Errorf("oops") })
	require.Error(t, err)
	assert.Equal(t, "oops", err.Error())
	assert.Equal(t, breaker.Open.String(), b.State().Status().String())

	// Confirm that it automatically returns an open error
	err = b.Run(func() error { return fmt.Errorf("oops") })
	require.Error(t, err)
	assert.Equal(t, breaker.ErrBreakerOpen, err)

	// Move the time to the timeout duration
	now = now.Add(time.Duration(10) * time.Second)
	assert.Equal(t, breaker.HalfOpen.String(), b.State().Status().String())

	// Send a single success to remain in half open state
	b.Run(func() error { return nil })
	assert.Equal(t, breaker.HalfOpen.String(), b.State().Status().String())

	// Force an error during half open state to trigger another timeout
	b.Run(func() error { return fmt.Errorf("oops") })
	assert.Equal(t, breaker.Open.String(), b.State().Status().String())

	// Move the time to the timeout duration again
	now = now.Add(time.Duration(10) * time.Second)
	assert.Equal(t, breaker.HalfOpen.String(), b.State().Status().String())

	// Send the necessary successes
	b.Run(func() error { return nil })
	b.Run(func() error { return nil })
	assert.Equal(t, breaker.Closed.String(), b.State().Status().String())
}

func TestWithPanic(t *testing.T) {
	b := breaker.New()
	err := b.Run(func() error { panic("oops") })
	require.Error(t, err)
	assert.Equal(t, "oops", err.Error())
}

func TestWithZeroValues(t *testing.T) {
	now := time.Now()

	b := breaker.New(
		breaker.WithNow(func() time.Time { return now }),
		breaker.WithIsFailure(nil),
		breaker.WithMaxHalfOpenRequests(0),
		breaker.WithMaxFailures(0),
		breaker.WithTimeout(0),
	)
	for i := 0; i < 4; i++ {
		b.Run(func() error { return fmt.Errorf("oops") })
	}

	// Not enough errors to trigger the open state
	assert.Equal(t, breaker.Closed.String(), b.State().Status().String())
	assert.Equal(t, 4, int(b.State().Failures))

	// Reaches the max failures to trigger the open state
	err := b.Run(func() error { return fmt.Errorf("oops") })
	require.Error(t, err)
	assert.Equal(t, "oops", err.Error())
	assert.Equal(t, breaker.Open.String(), b.State().Status().String())

	// Move the time to the timeout duration again
	now = now.Add(time.Duration(60) * time.Second)
	assert.Equal(t, breaker.HalfOpen.String(), b.State().Status().String())

	// Send a success request
	b.Run(func() error { return nil })
	assert.Equal(t, breaker.Closed.String(), b.State().Status().String())
}
