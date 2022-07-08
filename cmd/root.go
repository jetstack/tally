package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"cloud.google.com/go/bigquery"
	"github.com/ribbybibby/tally/internal/tally"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	All       bool
	Format    string
	Output    string
	ProjectID string
}

var ro rootOptions

var rootCmd = &cobra.Command{
	Use:   "tally",
	Short: "Finds OpenSSF Scorecard scores for packages in a Software Bill of Materials.",
	Long:  `Finds OpenSSF Scorecard scores for packages in a Software Bill of Materials.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		bq, err := bigquery.NewClient(ctx, ro.ProjectID)
		if err != nil {
			return err
		}

		var r io.ReadCloser
		if args[0] == "-" {
			r = os.Stdin
		} else {
			r, err = os.Open(args[0])
			if err != nil {
				return err
			}
			defer r.Close()
		}

		pkgs, err := tally.PackagesFromBOM(r, tally.BOMFormat(ro.Format))
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Found %d supported packages in BOM\n", len(pkgs))

		fmt.Fprintf(os.Stderr, "Retrieving scores from BigQuery...\n")
		pkgs, err = tally.ScorePackages(ctx, bq, pkgs)
		if err != nil {
			return err
		}

		return tally.WriteOutput(
			os.Stdout,
			pkgs,
			tally.WithOutput(tally.Output(ro.Output)),
			tally.WithOutputAll(ro.All),
		)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&ro.ProjectID, "project-id", "p", "", "Google Cloud project that Big Query requests are billed against")
	rootCmd.MarkFlagRequired("project-id")
	rootCmd.Flags().StringVarP(&ro.Format, "format", "f", string(tally.BOMFormatCycloneDXJSON), fmt.Sprintf("BOM format, options=%s", tally.BOMFormats))
	rootCmd.Flags().BoolVarP(&ro.All, "all", "a", false, "print all packages, even those without a scorecard score")
	rootCmd.Flags().StringVarP(&ro.Output, "output", "o", "short", fmt.Sprintf("output format, options=%s", tally.Outputs))
}
