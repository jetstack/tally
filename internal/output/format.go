package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

	// OutputFormatJSON prints the full report in JSON format.
	FormatJSON Format = "json"
)

// Formats are the supported output formats
var Formats = []Format{
	FormatShort,
	FormatWide,
	FormatJSON,
}

type writer func(io.Writer, types.Report) error

var writerMap = map[Format]writer{
	FormatShort: writeShort,
	FormatWide:  writeWide,
	FormatJSON:  writeJSON,
}

func writeShort(w io.Writer, report types.Report) error {
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
		} else {
			fmt.Fprintf(tw, "%s\t%s\n", result.Repository.Name, " ")
		}
		printed[result.Repository.Name] = struct{}{}
	}

	return nil
}

func writeWide(w io.Writer, report types.Report) error {
	tw := tabwriter.NewWriter(w, 0, 0, 1, ' ', 0)
	defer tw.Flush()
	fmt.Fprintf(tw, "TYPE\tPACKAGE\tREPOSITORY\tSCORE\n")

	for _, result := range report.Results {
		for _, pkg := range result.Packages {
			if result.Result != nil {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%.1f\n", pkg.Type, pkg.Name, result.Repository.Name, result.Result.Score)
			} else {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", pkg.Type, pkg.Name, result.Repository.Name, " ")
			}
		}
	}

	return nil
}

func writeJSON(w io.Writer, report types.Report) error {
	data, err := json.Marshal(report)
	if err != nil {
		return nil
	}

	if _, err := w.Write(data); err != nil {
		return err
	}

	return nil
}
