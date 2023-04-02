package timer_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agschwender/errcat-go/timer"
)

func TestTimer(t *testing.T) {
	var tmr *timer.Timer

	err := tmr.Run(func() error { return fmt.Errorf("oops") })
	require.NotNil(t, err)
	assert.Equal(t, "oops", err.Error())

	tmr = timer.New(time.Duration(50) * time.Millisecond)

	// Function returns an error
	err = tmr.Run(func() error { return fmt.Errorf("oops") })
	require.NotNil(t, err)
	assert.Equal(t, "oops", err.Error())

	// Function panics
	err = tmr.Run(func() error { panic("oops") })
	require.NotNil(t, err)
	assert.Equal(t, "oops", err.Error())

	// Function completes before timeout
	err = tmr.Run(func() error {
		time.Sleep(time.Duration(10) * time.Millisecond)
		return nil
	})
	assert.Nil(t, err)

	// Function takes longer than timeout
	err = tmr.Run(func() error {
		time.Sleep(time.Duration(75) * time.Millisecond)
		return nil
	})
	assert.Equal(t, timer.ErrTimeout, err)
}
