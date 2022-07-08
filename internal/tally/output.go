package tally

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"text/tabwriter"
)

// ErrUnsupportedOutput is returned when an output is requested by string that
// this package doesn't implement.
var ErrUnsupportedOutput = errors.New("unsupported output")

// Output is a supported output option
type Output string

const (
	// OutputShort prints the package system, name, version and score in
	// tab-separated columns.
	OutputShort Output = "short"

	// OutputWide prints the full range of information in tab-separated
	// columns.
	OutputWide Output = "wide"

	// OutputJSON prints the list of packages in JSON format.
	OutputJSON Output = "json"
)

// Outputs are the supported output options
var Outputs = []Output{
	OutputShort,
	OutputWide,
	OutputJSON,
}

type outputWriter func(io.Writer, []Package) error

var outputWriterMap = map[Output]outputWriter{
	OutputShort: writeShortOutput,
	OutputWide:  writeWideOutput,
	OutputJSON:  writeJSONOutput,
}

// WriteOutput writes packages to the provided io.Writer. The output can be
// configured by providing OutputOptions.
func WriteOutput(w io.Writer, pkgs []Package, opts ...OutputOption) error {
	o := &outputOptions{
		Output: OutputShort,
	}
	for _, opt := range opts {
		opt(o)
	}

	if !o.All {
		p := []Package{}
		for _, pkg := range pkgs {
			if pkg.RepositoryName == "" {
				continue
			}
			p = append(p, pkg)
		}
		pkgs = p
	}

	out, ok := outputWriterMap[o.Output]
	if !ok {
		return fmt.Errorf("%s: %w", o.Output, ErrUnsupportedOutput)
	}

	return out(w, pkgs)
}

// OutputOption is a functional option that configures the behaviour of outputs
type OutputOption func(*outputOptions)

type outputOptions struct {
	All    bool
	Output Output
}

// WithOutput is a functional option that configures the kind of output
func WithOutput(output Output) OutputOption {
	return func(o *outputOptions) {
		o.Output = output
	}
}

// WithOutputAll is a functional option that configures outputs to return all
// packages, even if they don't have a scorecard score
func WithOutputAll(all bool) OutputOption {
	return func(o *outputOptions) {
		o.All = all
	}
}

func writeShortOutput(w io.Writer, pkgs []Package) error {
	tw := tabwriter.NewWriter(w, 0, 0, 1, ' ', 0)
	defer tw.Flush()
	fmt.Fprintf(tw, "SYSTEM\tPACKAGE\tVERSION\tSCORE\n")

	for _, pkg := range pkgs {
		if pkg.Score > 0 {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%.1f\n", pkg.System, pkg.Name, pkg.Version, pkg.Score)
		} else {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", pkg.System, pkg.Name, pkg.Version, " ")
		}
	}

	return nil
}

func writeWideOutput(w io.Writer, pkgs []Package) error {
	tw := tabwriter.NewWriter(w, 0, 0, 1, ' ', 0)
	defer tw.Flush()
	fmt.Fprintf(tw, "SYSTEM\tPACKAGE\tVERSION\tREPOSITORY\tSCORE\tDATE\n")

	for _, pkg := range pkgs {
		if pkg.Score > 0 {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%.1f\t%s\n", pkg.System, pkg.Name, pkg.Version, pkg.RepositoryName, pkg.Score, pkg.Date)
		} else {

			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", pkg.System, pkg.Name, pkg.Version, pkg.RepositoryName, " ", " ")
		}
	}

	return nil
}

func writeJSONOutput(w io.Writer, pkgs []Package) error {
	data, err := json.Marshal(pkgs)
	if err != nil {
		return nil
	}

	if _, err := w.Write(data); err != nil {
		return err
	}

	return nil
}
