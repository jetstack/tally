package cache

import "time"

// Option is a functional option that configures a cache
type Option func(o *options)

type options struct {
	Duration time.Duration
}

func makeOptions(opts ...Option) *options {
	o := &options{
		Duration: (24 * time.Hour) * 7,
	}
	for _, opt := range opts {
		opt(o)
	}

	return o
}

// WithDuration is a functional option that configures the amount of time until
// an item in the cache is invalidated
func WithDuration(d time.Duration) Option {
	return func(o *options) {
		o.Duration = d
	}
}
