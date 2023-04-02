package fallback_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/agschwender/errcat-go/fallback"
)

func TestAsNil(t *testing.T) {
	var f *fallback.Fallback

	assert.False(t, f.UseFallback(fmt.Errorf("oops")))
	assert.False(t, f.UseFallback(nil))
	assert.Nil(t, f.Call())
}

func TestWithDefaults(t *testing.T) {
	f := fallback.New(nil)

	assert.True(t, f.UseFallback(fmt.Errorf("oops")))
	assert.False(t, f.UseFallback(nil))
	assert.Nil(t, f.Call())
}

func TestWithOverrides(t *testing.T) {
	counts := 0
	f := fallback.New(func() error {
		counts++
		return nil
	}, fallback.WithUseFallback(func(err error) bool {
		return err != nil && err.Error() != "no fallback"
	}))

	assert.True(t, f.UseFallback(fmt.Errorf("oops")))
	assert.False(t, f.UseFallback(fmt.Errorf("no fallback")))
	assert.False(t, f.UseFallback(nil))
	assert.Nil(t, f.Call())
	assert.Equal(t, 1, counts)
}

func TestWithZeroValues(t *testing.T) {
	f := fallback.New(nil, fallback.WithUseFallback(nil))

	assert.True(t, f.UseFallback(fmt.Errorf("oops")))
	assert.False(t, f.UseFallback(nil))
	assert.Nil(t, f.Call())
}
