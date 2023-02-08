package output

import (
	"io"
	"sort"

	"github.com/jetstack/tally/internal/types"
)

// Output writes output for tally
type Output interface {
	WriteResults(io.Writer, []types.Result) error
}

// NewOutput returns a new output, configured by the provided options
func NewOutput(opts ...Option) (Output, error) {
	o := &output{
		writer: writeShort,
	}
	for _, opt := range opts {
		if err := opt(o); err != nil {
			return nil, err
		}
	}

	return o, nil
}

type output struct {
	all    bool
	writer writer
}

// WriteResults writes the provided results to the given io.Writer in the
// configured output format
func (o *output) WriteResults(w io.Writer, results []types.Result) error {
	// Unless -a is configured, ignore packages without a score
	if !o.all {
		r := []types.Result{}
		for _, result := range results {
			if result.Score == nil {
				continue
			}
			r = append(r, result)
		}
		results = r
	}

	// Sort the packages by score in the output
	sort.Slice(results, func(i, j int) bool {
		var (
			is float64
			js float64
		)
		if results[i].Score != nil {
			is = results[i].Score.Score
		}

		if results[j].Score != nil {
			js = results[j].Score.Score
		}

		// If the scores are equal, then sort by repository.
		if is == js {
			return results[i].Repository > results[j].Repository
		}

		return is > js
	})

	return o.writer(w, results)
}
