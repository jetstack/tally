package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
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

	// FormatChecks prints the individual check scores for each repository
	FormatChecks Format = "checks"

	// OutputFormatJSON prints the list of packages in JSON format.
	FormatJSON Format = "json"
)

// Formats are the supported output formats
var Formats = []Format{
	FormatShort,
	FormatWide,
	FormatChecks,
	FormatJSON,
}

type writer func(io.Writer, []types.Result) error

var writerMap = map[Format]writer{
	FormatShort:  writeShort,
	FormatWide:   writeWide,
	FormatChecks: writeChecks,
	FormatJSON:   writeJSON,
}

func writeShort(w io.Writer, results []types.Result) error {
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
		if result.Score != nil {
			fmt.Fprintf(tw, "%s\t%.1f\n", result.Repository, result.Score.Score)
		} else {
			fmt.Fprintf(tw, "%s\t%s\n", result.Repository, " ")
		}
		printed[result.Repository] = struct{}{}
	}

	return nil
}

func writeWide(w io.Writer, results []types.Result) error {
	tw := tabwriter.NewWriter(w, 0, 0, 1, ' ', 0)
	defer tw.Flush()
	fmt.Fprintf(tw, "SYSTEM\tPACKAGE\tREPOSITORY\tSCORE\n")

	for _, result := range results {
		if result.Score != nil {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%.1f\n", result.PackageSystem, result.PackageName, result.Repository, result.Score.Score)
		} else {

			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", result.PackageSystem, result.PackageName, result.Repository, " ")
		}
	}

	return nil
}

func writeChecks(w io.Writer, results []types.Result) error {
	var checks []string
	for _, result := range results {
		if result.Score == nil {
			continue
		}
		for n := range result.Score.Checks {
			if contains(checks, n) {
				continue
			}

			checks = append(checks, n)
		}
	}
	sort.Strings(checks)

	header := "REPOSITORY\tSCORE"
	for _, check := range checks {
		header = header + fmt.Sprintf("\t%s", strings.ToUpper(check))
	}
	header = header + "\n"

	tw := tabwriter.NewWriter(w, 0, 0, 1, ' ', 0)
	defer tw.Flush()

	fmt.Fprintf(tw, header)

	printed := map[string]struct{}{}
	for _, result := range results {
		if result.Repository == "" {
			continue
		}
		if _, ok := printed[result.Repository]; ok {
			continue
		}
		var line string
		if result.Score != nil {
			line = fmt.Sprintf("%s\t%.1f", result.Repository, result.Score.Score)
			scores := make([]int, len(checks))
			for n, score := range result.Score.Checks {
				for i, check := range checks {
					if n != check {
						continue
					}
					scores[i] = score
				}
			}
			for _, score := range scores {
				line = line + fmt.Sprintf("\t%d", score)
			}
		} else {
			line = fmt.Sprintf("%s\t%s", result.Repository, "")
			for range checks {
				line = line + "\t "
			}
		}
		line = line + "\n"

		fmt.Fprintf(tw, line)

		printed[result.Repository] = struct{}{}
	}

	return nil
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}

	return false
}

func writeJSON(w io.Writer, results []types.Result) error {
	data, err := json.Marshal(results)
	if err != nil {
		return nil
	}

	if _, err := w.Write(data); err != nil {
		return err
	}

	return nil
}
