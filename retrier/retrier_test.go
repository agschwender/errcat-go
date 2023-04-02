package retrier_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agschwender/errcat-go/retrier"
)

func TestAsNil(t *testing.T) {
	var r *retrier.Retrier

	counts := 0
	err := r.Run(func() error {
		counts++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, counts)
}

func TestWithDefaults(t *testing.T) {
	// Confirm happy path
	counts := 0
	err := retrier.New().Run(func() error {
		counts++
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 1, counts)

	// Confirm attempts with error
	counts = 0
	err = retrier.New().Run(func() error {
		counts++
		return fmt.Errorf("oops")
	})
	require.Error(t, err)
	assert.Equal(t, 1, counts)
}

func TestWithOverrides(t *testing.T) {
	r := retrier.New(
		retrier.WithIsRetriable(func(err error) bool {
			return err == nil || err.Error() != "perm err"
		}),
		retrier.WithMaxAttempts(3),
	)

	// Confirm attempts with error
	counts := 0
	err := r.Run(func() error {
		counts++
		return fmt.Errorf("oops")
	})
	require.Error(t, err)
	assert.Equal(t, 3, counts)

	// Confirm exit on non-retriable failure
	counts = 0
	err = r.Run(func() error {
		counts++
		if counts == 2 {
			return fmt.Errorf("perm err")
		}
		return fmt.Errorf("oops")
	})
	require.Error(t, err)
	assert.Equal(t, "perm err", err.Error())
	assert.Equal(t, 2, counts)
}

func TestWithZeroValues(t *testing.T) {
	r := retrier.New(
		retrier.WithIsRetriable(nil),
		retrier.WithMaxAttempts(0),
	)

	// Confirm attempts with error
	counts := 0
	err := r.Run(func() error {
		counts++
		return fmt.Errorf("oops")
	})
	require.Error(t, err)
	assert.Equal(t, 1, counts)
}
