package errcat_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/agschwender/errcat-go"
	"github.com/agschwender/errcat-go/breaker"
	"github.com/agschwender/errcat-go/fallback"
	"github.com/agschwender/errcat-go/retrier"
)

func TestCallerWithDefaults(t *testing.T) {
	counts := 0
	err := errcat.New("mysql", "users.GetUser").Call(func() error {
		counts++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, counts)
}

func TestCallerWithOptions(t *testing.T) {
	fallbacks := 0
	c := errcat.New("google", "clients.Google.Search").
		WithBreaker(breaker.New()).
		WithFallback(fallback.New(func() error {
			fallbacks++
			return nil
		})).
		WithRetrier(retrier.New(retrier.WithMaxAttempts(3))).
		WithTimeout(time.Duration(1) * time.Second)

	counts := 0
	err := c.Call(func() error {
		counts++
		return fmt.Errorf("oops")
	})
	assert.NoError(t, err)
	assert.Equal(t, 3, counts)
	assert.Equal(t, 1, fallbacks)

}
