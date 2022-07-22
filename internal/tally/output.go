package tally

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"
)

// ErrUnsupportedOutputFormat is returned when an output is requested by string that
// this package doesn't implement.
var ErrUnsupportedOutputFormat = errors.New("unsupported output")

// OutputFormat is a supported output option
type OutputFormat string

const (
	// OutputFormatShort prints the repositories and their scores.
	OutputFormatShort OutputFormat = "short"

	// OutputFormatWide prints the package version information, as well as
	// the repositories and their scores.
	OutputFormatWide OutputFormat = "wide"

	// OutputFormatJSON prints the list of packages in JSON format.
	OutputFormatJSON OutputFormat = "json"
)

// OutputFormats are the supported output options
var OutputFormats = []OutputFormat{
	OutputFormatShort,
	OutputFormatWide,
	OutputFormatJSON,
}

// Output writes output for tally
type Output interface {
	WriteResults(io.Writer, []Result) error
}

// NewOutput returns a new output, configured by the provided options
func NewOutput(opts ...OutputOption) (Output, error) {
	o := &output{
		writer: writeShortOutput,
	}
	for _, opt := range opts {
		if err := opt(o); err != nil {
			return nil, err
		}
	}

	return o, nil
}

// OutputOption is a functional option that configures the behaviour of output
type OutputOption func(*output) error

type output struct {
	all    bool
	writer outputWriter
}

// WriteResults writes the provided results to the given io.Writer in the
// configured output format
func (o *output) WriteResults(w io.Writer, results []Result) error {
	// Unless -a is configured, ignore packages without a score
	if !o.all {
		r := []Result{}
		for _, result := range results {
			if result.Score == 0 {
				continue
			}
			r = append(r, result)
		}
		results = r
	}

	// Sort the packages by score in the output
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return o.writer(w, results)
}

// WithOutputFormat is a functional option that configures the format of the
// output
func WithOutputFormat(format OutputFormat) OutputOption {
	return func(o *output) error {
		writer, ok := outputWriterMap[format]
		if !ok {
			return fmt.Errorf("%s: %w", format, ErrUnsupportedOutputFormat)
		}
		o.writer = writer

		return nil
	}
}

// WithOutputAll is a functional option that configures outputs to return all
// packages, even if they don't have a scorecard score
func WithOutputAll(all bool) OutputOption {
	return func(o *output) error {
		o.all = all

		return nil
	}
}

type outputWriter func(io.Writer, []Result) error

var outputWriterMap = map[OutputFormat]outputWriter{
	OutputFormatShort: writeShortOutput,
	OutputFormatWide:  writeWideOutput,
	OutputFormatJSON:  writeJSONOutput,
}

func writeShortOutput(w io.Writer, results []Result) error {
	tw := tabwriter.NewWriter(w, 0, 0, 1, ' ', 0)
	defer tw.Flush()
	fmt.Fprintf(tw, "REPOSITORY\tSCORE\n")

	printed := map[string]struct{}{}
	for _, result := range results {
		if result.Repository == "" {
			continue
		}
		if _, ok := printed[result.Repository]; ok {
			continue
		}
		if result.Score > 0 {
			fmt.Fprintf(tw, "%s\t%.1f\n", result.Repository, result.Score)
		} else {
			fmt.Fprintf(tw, "%s\t%s\n", result.Repository, " ")
		}
		printed[result.Repository] = struct{}{}
	}

	return nil
}

func writeWideOutput(w io.Writer, results []Result) error {
	tw := tabwriter.NewWriter(w, 0, 0, 1, ' ', 0)
	defer tw.Flush()
	fmt.Fprintf(tw, "TYPE\tPACKAGE\tVERSION\tREPOSITORY\tTABLE\tSCORE\tDATE\n")

	for _, result := range results {
		if result.Score > 0 {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%.1f\t%s\n", result.Package.Type, result.Package.Name, result.Package.Version, result.Repository, result.Table, result.Score, result.Date)
		} else {

			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", result.Package.Type, result.Package.Name, result.Package.Version, result.Repository, result.Table, " ", " ")
		}
	}

	return nil
}

func writeJSONOutput(w io.Writer, results []Result) error {
	data, err := json.Marshal(results)
	if err != nil {
		return nil
	}

	if _, err := w.Write(data); err != nil {
		return err
	}

	return nil
}
