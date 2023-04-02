package retrier

const defaultMaxAttempts = uint(1)

var defaultIsRetriable = func(err error) bool { return err != nil }

type Retrier struct {
	isRetriable func(err error) bool
	maxAttempts uint
}

type option func(*Retrier)

// New creates a new Retrier with the supplied options.
func New(opts ...option) *Retrier {
	r := &Retrier{
		isRetriable: defaultIsRetriable,
		maxAttempts: defaultMaxAttempts,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// WithIsRetriable defines the logic for determining if the error should
// be retried.
func WithIsRetriable(isRetriable func(err error) bool) option {
	return func(r *Retrier) {
		if isRetriable == nil {
			isRetriable = defaultIsRetriable
		}
		r.isRetriable = isRetriable
	}
}

// WithMaxAttempts defines the maximum number of attempts the retrier
// can make.
func WithMaxAttempts(maxAttempts uint) option {
	return func(r *Retrier) {
		if maxAttempts == 0 {
			maxAttempts = defaultMaxAttempts
		}
		r.maxAttempts = maxAttempts
	}
}

// Run executes the callback until it succeeds or the maximum number of
// attempts is reached.
func (r *Retrier) Run(cb func() error) error {
	if r == nil {
		return cb()
	}

	var err error
	for i := uint(0); i < r.maxAttempts; i++ {
		err = cb()
		if err == nil || !r.isRetriable(err) {
			return err
		}
	}
	return err
}
