package fallback

var defaultUseFallback = func(err error) bool { return err != nil }

type FallbackFn func() error

type Fallback struct {
	useFallback func(err error) bool
	call        FallbackFn
}

type option func(*Fallback)

// New creates a new Fallback with the supplied options.
func New(call FallbackFn, opts ...option) *Fallback {
	if call == nil {
		call = func() error { return nil }
	}

	f := &Fallback{
		useFallback: defaultUseFallback,
		call:        call,
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// WithUseFallback sets those errors that should trigger a fallback.
func WithUseFallback(useFallback func(err error) bool) option {
	return func(f *Fallback) {
		if useFallback == nil {
			useFallback = defaultUseFallback
		}
		f.useFallback = useFallback
	}
}

// UseFallback will indicate if the fallback should be called.
func (f *Fallback) UseFallback(err error) bool {
	return f != nil && f.useFallback(err)
}

// Call will execute the fallback function. Callers should check
// UseFallback prior to calling.
func (f *Fallback) Call() error {
	if f == nil {
		return nil
	}
	return f.call()
}
