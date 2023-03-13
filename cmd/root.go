package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jetstack/tally/internal/bom"
	"github.com/jetstack/tally/internal/cache"
	"github.com/jetstack/tally/internal/output"
	"github.com/jetstack/tally/internal/scorecard"
	scorecardapi "github.com/jetstack/tally/internal/scorecard/api"
	"github.com/jetstack/tally/internal/tally"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	All            bool
	API            bool
	APITimeout     time.Duration
	APIURL         string
	Cache          bool
	CacheDir       string
	CacheDuration  time.Duration
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
		pkgs, err := bom.PackagesFromBOM(r, bom.Format(ro.Format))
		if err != nil {
			return err
		}

		var scorecardClients []scorecard.Client

		// Fetch scores from the API
		if ro.API {
			apiClient, err := scorecardapi.NewClient(ro.APIURL)
			if err != nil {
				return fmt.Errorf("configuring API client: %w", err)
			}
			scorecardClients = append(scorecardClients, apiClient)
		}

		// Generate scores with the scorecard client
		if ro.GenerateScores {
			sc, err := scorecard.NewScorecardClient()
			if err != nil {
				return fmt.Errorf("configuring scorecard client: %w", err)
			}
			scorecardClients = append(scorecardClients, sc)
		}

		// At least one scorecard client must be configured
		if len(scorecardClients) == 0 {
			fmt.Fprintf(os.Stderr, "Error: no scorecard clients configured. At least one of --api or --generate must be set.\n")
			os.Exit(1)
		}

		// Cache scorecard results locally to speed up subsequent runs
		if ro.Cache {
			dbCache, err := cache.NewSqliteCache(ro.CacheDir, cache.WithDuration(ro.CacheDuration))
			if err != nil {
				return fmt.Errorf("creating cache: %w", err)
			}

			// Wrap our clients with the cache
			for i, client := range scorecardClients {
				scorecardClients[i] = cache.NewScorecardClient(dbCache, client)
			}
		}

		// Get results
		results, err := tally.Results(ctx, os.Stderr, scorecardClients, pkgs...)
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
				if result.ScorecardResult == nil || result.ScorecardResult.Score > *ro.FailOn.Value {
					continue
				}
				fmt.Fprintf(os.Stderr, "Error: found scores <= to %0.2f\n", *ro.FailOn.Value)
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
	rootCmd.Flags().BoolVar(&ro.API, "api", true, "fetch scores from the Scorecard API")
	rootCmd.Flags().DurationVar(&ro.APITimeout, "api-timeout", scorecardapi.DefaultTimeout, "timeout for requests to scorecard API")
	rootCmd.Flags().StringVar(&ro.APIURL, "api-url", scorecardapi.DefaultURL, "scorecard API URL")
	rootCmd.Flags().StringVarP(&ro.Output, "output", "o", "short", fmt.Sprintf("output format, options=%s", output.Formats))
	rootCmd.Flags().BoolVarP(&ro.GenerateScores, "generate", "g", false, "generate scores for repositories that aren't in the database. The GITHUB_TOKEN environment variable must be set.")
	rootCmd.Flags().BoolVar(&ro.Cache, "cache", true, "cache scores locally")
	rootCmd.Flags().StringVar(&ro.CacheDir, "cache-dir", "", "directory to cache scores in, defaults to $HOME/.cache/tally/cache on most systems")
	rootCmd.Flags().DurationVar(&ro.CacheDuration, "cache-duration", 7*(24*time.Hour), "how long to cache scores for; defaults to 7 days")
	rootCmd.Flags().Var(&ro.FailOn, "fail-on", "fail if a package is found with a score <= to the given value")
}
