package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/jetstack/tally/internal/bom"
	"github.com/jetstack/tally/internal/manager"
	"github.com/jetstack/tally/internal/output"
	"github.com/jetstack/tally/internal/repositories"
	"github.com/jetstack/tally/internal/scorecard"
	scorecarddb "github.com/jetstack/tally/internal/scorecard/db"
	"github.com/jetstack/tally/internal/tally"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	All            bool
	CheckForUpdate bool
	DBRef          string
	FailOn         float64Flag
	Format         string
	GenerateScores bool
	Output         string
}

var ro rootOptions

var rootCmd = &cobra.Command{
	Use:   "tally",
	Short: "Finds OpenSSF Scorecard scores for packages in a Software Bill of Materials.",
	Long:  `Finds OpenSSF Scorecard scores for packages in a Software Bill of Materials.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := context.Background()

		// Configure the output writer
		out, err := output.NewOutput(
			output.WithFormat(output.Format(ro.Output)),
			output.WithAll(ro.All),
		)
		if err != nil {
			return fmt.Errorf("creating output writer: %w", err)
		}

		mgr, err := manager.NewManager(manager.WithWriter(os.Stderr))
		if err != nil {
			return fmt.Errorf("creating database manager: %w", err)
		}

		// Update the database, if there's a new version available
		if ro.CheckForUpdate {
			updated, err := mgr.PullDB(ctx, ro.DBRef)
			if err != nil {
				return fmt.Errorf("importing database: %w", err)
			}
			if updated {
				fmt.Fprintf(os.Stderr, "Pulled database.\n")
			}
		}

		tallyDB, err := mgr.DB()
		if err != nil {
			return fmt.Errorf("getting database from manager: %w", err)
		}
		defer tallyDB.Close()

		// Get packages from the BOM
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
		sbom, err := bom.ParseBOM(r, bom.Format(ro.Format))
		if err != nil {
			return err
		}
		pkgs, err := sbom.Packages()
		if err != nil {
			return err
		}

		// Map packages to repositories, using various different sources
		repoMapper := repositories.From(
			repositories.PackageMapper,
			repositories.BOMMapper(sbom),
			repositories.DBMapper(tallyDB),
		)

		// Fetch scores from the database
		scorecardClients := []scorecard.Client{
			scorecarddb.NewClient(tallyDB),
		}

		// Optionally generate scores with the scorecard client
		if ro.GenerateScores {
			sc, err := scorecard.NewScorecardClient()
			if err != nil {
				return fmt.Errorf("configuring scorecard client: %w", err)
			}
			scorecardClients = append(scorecardClients, sc)
		}

		// Get results
		results, err := tally.Results(ctx, os.Stderr, repoMapper, scorecardClients, pkgs...)
		if err != nil {
			return fmt.Errorf("getting results: %w", err)
		}

		// Write results to output
		if err := out.WriteResults(os.Stdout, results); err != nil {
			os.Exit(1)
		}

		// Exit 1 if there is a score <= o.FailOn
		if ro.FailOn.Value != nil {
			for _, result := range results {
				if result.Score == nil || result.Score.Score > *ro.FailOn.Value {
					continue
				}
				fmt.Fprintf(os.Stderr, "error: found scores <= to %0.2f\n", *ro.FailOn.Value)
				os.Exit(1)
			}
		}

		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&ro.Format, "format", "f", string(bom.FormatCycloneDXJSON), fmt.Sprintf("BOM format, options=%s", bom.Formats))
	rootCmd.Flags().BoolVarP(&ro.All, "all", "a", false, "print all packages, even those without a scorecard score")
	rootCmd.Flags().StringVarP(&ro.Output, "output", "o", "short", fmt.Sprintf("output format, options=%s", output.Formats))
	rootCmd.Flags().BoolVarP(&ro.GenerateScores, "generate", "g", false, "generate scores for repositories that aren't in the database. The GITHUB_TOKEN environment variable must be set.")
	rootCmd.Flags().BoolVar(&ro.CheckForUpdate, "check-for-update", true, "check for database update")
	rootCmd.Flags().StringVar(&ro.DBRef, "db-reference", "ghcr.io/jetstack/tally/db:v1", "image reference to download database from")
	rootCmd.Flags().Var(&ro.FailOn, "fail-on", "fail if a package is found with a score <= to the given value")
}
