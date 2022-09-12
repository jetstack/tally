package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"cloud.google.com/go/bigquery"
	"github.com/jetstack/tally/internal/bom"
	"github.com/jetstack/tally/internal/db"
	"github.com/jetstack/tally/internal/db/db/local"
	"github.com/jetstack/tally/internal/output"
	"github.com/jetstack/tally/internal/scorecard"
	"github.com/jetstack/tally/internal/tally"
	"github.com/jetstack/tally/internal/types"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	All            bool
	Dataset        string
	FailOn         float64Flag
	Format         string
	GenerateScores bool
	Output         string
	ProjectID      string
}

var ro rootOptions

var rootCmd = &cobra.Command{
	Use:   "tally",
	Short: "Finds OpenSSF Scorecard scores for packages in a Software Bill of Materials.",
	Long:  `Finds OpenSSF Scorecard scores for packages in a Software Bill of Materials.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if ro.GenerateScores && os.Getenv("GITHUB_TOKEN") == "" {
			return fmt.Errorf("must set GITHUB_TOKEN environment variable with -g/--generate")
		}

		ctx := context.Background()

		out, err := output.NewOutput(
			output.WithFormat(output.Format(ro.Output)),
			output.WithAll(ro.All),
		)
		if err != nil {
			return err
		}

		var table scorecard.Table
		if ro.Dataset != "" {
			if ro.ProjectID == "" {
				return fmt.Errorf("must set --project-id with --dataset")
			}
			bq, err := bigquery.NewClient(ctx, ro.ProjectID)
			if err != nil {
				return err
			}
			dataset, err := tally.NewDataset(bq, ro.Dataset)
			if err != nil {
				return err
			}
			table, err = dataset.ScorecardTable()
			if err != nil {
				return err
			}
		}

		dbPath, err := local.Path()
		if err != nil {
			return fmt.Errorf("getting database path: %w", err)
		}

		if _, err := os.Stat(dbPath); err != nil {
			return fmt.Errorf("statting db  %q: %w", dbPath, err)
		}

		tallyDB, err := local.NewDB(dbPath)
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
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

		// Get packages from BOM
		pkgs, err := bom.PackagesFromBOM(r, bom.Format(ro.Format))
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Found %d supported packages in BOM\n", len(pkgs))

		// Find repositories for the packages
		for i, pkg := range pkgs {
			repos, err := tallyDB.GetRepositories(ctx, pkg.System, pkg.Name)
			if err != nil {
				if err != db.ErrNotFound {
					return err
				}
				continue
			}

			for _, repo := range repos {
				if !contains(pkg.Repositories, repo) {
					pkgs[i].Repositories = append(pkgs[i].Repositories, repo)
				}
			}
		}

		// Get score for each package+repository combination
		var results []types.Result
		for _, pkg := range pkgs {
			if len(pkg.Repositories) == 0 {
				results = append(results, types.Result{
					PackageSystem: pkg.System,
					PackageName:   pkg.Name,
				})
				continue
			}
			for _, repo := range pkg.Repositories {
				result := types.Result{
					PackageSystem: pkg.System,
					PackageName:   pkg.Name,
					Repository:    repo,
				}

				score, err := getScore(ctx, tallyDB, repo)
				if err != nil && err != db.ErrNotFound {
					return err
				}
				result.Score = score

				results = append(results, result)
			}
		}

		// Integrate scores from a private scorecard dataset
		if table != nil {
			fmt.Fprintf(os.Stderr, "Fetching scores from %s...\n", table)
			results, err = tally.AddScoresFromScorecardTable(ctx, table, results)
			if err != nil {
				return err
			}
		}

		// Generate missing scores, write them to a private dataset
		if ro.GenerateScores {
			fmt.Fprintf(os.Stderr, "Generating missing scores...\n")
			results, err = tally.GenerateScores(ctx, table, results)
			if err != nil {
				return err
			}
		}

		// Write output
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

func getScore(ctx context.Context, tallyDB db.DB, repo string) (*types.Score, error) {
	var score *types.Score

	// Get the overall score
	s, err := tallyDB.GetScore(ctx, repo)
	if err != nil {
		return nil, err
	}
	score = &types.Score{
		Score:  s,
		Checks: map[string]int{},
	}

	// Get the individual check scores
	checks, err := tallyDB.GetChecks(ctx, repo)
	if err != nil {
		return nil, err
	}
	for _, check := range checks {
		score.Checks[check.Name] = check.Score
	}

	return score, nil
}

func contains(vals []string, val string) bool {
	for _, v := range vals {
		if v == val {
			return true
		}
	}

	return false
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&ro.ProjectID, "project-id", "p", "", "Google Cloud project that Big Query requests are billed against")
	rootCmd.Flags().StringVarP(&ro.Format, "format", "f", string(bom.FormatCycloneDXJSON), fmt.Sprintf("BOM format, options=%s", bom.Formats))
	rootCmd.Flags().BoolVarP(&ro.All, "all", "a", false, "print all packages, even those without a scorecard score")
	rootCmd.Flags().StringVarP(&ro.Output, "output", "o", "short", fmt.Sprintf("output format, options=%s", output.Formats))
	rootCmd.Flags().BoolVarP(&ro.GenerateScores, "generate", "g", false, "generate scores for repositories that aren't in the public dataset. The GITHUB_TOKEN environment variable must be set.")
	rootCmd.Flags().StringVarP(&ro.Dataset, "dataset", "d", "", "dataset for generated scores")
	rootCmd.Flags().Var(&ro.FailOn, "fail-on", "fail if a package is found with a score <= to the given value")
}
