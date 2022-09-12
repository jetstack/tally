package output

import (
	"fmt"
)

// Option is a functional option that configures the output behaviour
type Option func(*output) error

// WithFormat is a functional option that configures the format of the
// output
func WithFormat(format Format) Option {
	return func(o *output) error {
		writer, ok := writerMap[format]
		if !ok {
			return fmt.Errorf("%s: %w", format, ErrUnsupportedFormat)
		}
		o.writer = writer

		return nil
	}
}

// WithAll is a functional option that configures outputs to return all
// packages, even if they don't have a scorecard score
func WithAll(all bool) Option {
	return func(o *output) error {
		o.all = all

		return nil
	}
}
