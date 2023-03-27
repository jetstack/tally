package output

import (
	"io"
	"sort"

	"github.com/jetstack/tally/internal/types"
)

// Output writes output for tally
type Output interface {
	WriteReport(io.Writer, types.Report) error
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

// WriteReport writes the report to the given io.Writer in the
// configured output format
func (o *output) WriteReport(w io.Writer, report types.Report) error {
	// Unless -a is configured, ignore packages without a score
	if !o.all {
		r := []types.Result{}
		for _, result := range report.Results {
			if result.Result == nil {
				continue
			}
			r = append(r, result)
		}
		report.Results = r
	}

	// Sort the results by score
	sort.Slice(report.Results, func(i, j int) bool {
		var (
			is float64
			js float64
		)
		if report.Results[i].Result != nil {
			is = report.Results[i].Result.Score
		}

		if report.Results[j].Result != nil {
			js = report.Results[j].Result.Score
		}

		// If the scores are equal, then sort by repository.name
		if is == js {
			return report.Results[i].Repository.Name > report.Results[j].Repository.Name
		}

		return is > js
	})

	return o.writer(w, report)
}
