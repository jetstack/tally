package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/jetstack/tally/internal/types"
)

// ErrUnsupportedOutputFormat is returned when an output is requested by string that
// this package doesn't implement.
var ErrUnsupportedFormat = errors.New("unsupported output")

// Format is a supported output format
type Format string

const (
	// FormatShort prints the repositories and their scores.
	FormatShort Format = "short"

	// FormatWide prints the package version information, as well as
	// the repositories and their scores.
	FormatWide Format = "wide"

	// FormatJSON prints the report as a JSON document
	FormatJSON Format = "json"
)

// Formats are the supported output formats
var Formats = []Format{
	FormatShort,
	FormatWide,
	FormatJSON,
}

// Option is a functional option that configures the output behaviour
type Option func(*output)

// WithAll is a functional option that configures outputs to return all
// packages, even if they don't have a scorecard score
func WithAll(all bool) Option {
	return func(o *output) {
		o.all = all
	}
}

// Output writes output for tally
type Output interface {
	WriteReport(io.Writer, types.Report) error
}

// NewOutput returns a new output, configured by the provided options
func NewOutput(format Format, opts ...Option) (Output, error) {
	o := &output{}
	for _, opt := range opts {
		opt(o)
	}

	switch format {
	case FormatShort:
		o.writer = o.writeShort
	case FormatWide:
		o.writer = o.writeWide
	case FormatJSON:
		o.writer = o.writeJSON
	default:
		return nil, fmt.Errorf("%s: %w", format, ErrUnsupportedFormat)
	}

	return o, nil
}

type output struct {
	all    bool
	writer func(io.Writer, types.Report) error
}

// WriteReport writes the report to the given io.Writer in the
// configured output format
func (o *output) WriteReport(w io.Writer, report types.Report) error {
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

func (o *output) writeShort(w io.Writer, report types.Report) error {
	tw := tabwriter.NewWriter(w, 0, 0, 1, ' ', 0)
	defer tw.Flush()
	fmt.Fprintf(tw, "REPOSITORY\tSCORE\n")

	printed := map[string]struct{}{}
	for _, result := range report.Results {
		if result.Repository.Name == "" {
			continue
		}
		if _, ok := printed[result.Repository.Name]; ok {
			continue
		}
		if result.Result != nil {
			fmt.Fprintf(tw, "%s\t%.1f\n", result.Repository.Name, result.Result.Score)
		} else if o.all {
			fmt.Fprintf(tw, "%s\t%s\n", result.Repository.Name, " ")
		}
		printed[result.Repository.Name] = struct{}{}
	}

	return nil
}

func (o *output) writeWide(w io.Writer, report types.Report) error {
	tw := tabwriter.NewWriter(w, 0, 0, 1, ' ', 0)
	defer tw.Flush()
	fmt.Fprintf(tw, "TYPE\tPACKAGE\tREPOSITORY\tSCORE\n")

	for _, result := range report.Results {
		for _, pkg := range result.Packages {
			if result.Result != nil {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%.1f\n", pkg.Type, pkg.Name, result.Repository.Name, result.Result.Score)
			} else if o.all {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", pkg.Type, pkg.Name, result.Repository.Name, " ")
			}
		}
	}

	return nil
}

func (o *output) writeJSON(w io.Writer, report types.Report) error {
	data, err := json.Marshal(report)
	if err != nil {
		return nil
	}

	if _, err := w.Write(data); err != nil {
		return err
	}

	return nil
}
