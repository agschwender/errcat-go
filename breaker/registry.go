package breaker

import (
	"fmt"
)

// Registry stores circuit breakers by name. This is useful because a
// circuit breaker must be re-used for each call of the same type and
// the registry provides a mechanism for retrieving that circuit
// breaker.
type Registry map[string]Breaker

// Creates a new circuit breaker registry.
func NewRegistry() Registry {
	return make(map[string]Breaker)
}

// Register associates the supplied circuit breaker with the name and
// stores in the registry.
func (r Registry) Register(name string, b Breaker) error {
	if _, ok := r[name]; ok {
		return fmt.Errorf("breaker already registered with the name of %q", name)
	}
	r[name] = b
	return nil
}

// Gets the supplied circuit breaker using its name. The second return
// value indicates whether it was found.
func (r Registry) Get(name string) (Breaker, bool) {
	b, ok := r[name]
	return b, ok
}
