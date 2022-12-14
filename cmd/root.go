package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/jetstack/tally/internal/bom"
	"github.com/jetstack/tally/internal/db"
	"github.com/jetstack/tally/internal/manager"
	"github.com/jetstack/tally/internal/output"
	"github.com/jetstack/tally/internal/scorecard"
	"github.com/jetstack/tally/internal/types"
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
		if ro.GenerateScores && os.Getenv("GITHUB_TOKEN") == "" {
			return fmt.Errorf("must set GITHUB_TOKEN environment variable with -g/--generate")
		}

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
		pkgs, err := bom.PackagesFromBOM(r, bom.Format(ro.Format))
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Found %d supported packages in BOM\n", len(pkgs))

		// Find repositories for the packages from the database
		for i, pkg := range pkgs {
			repos, err := tallyDB.GetRepositories(ctx, pkg.System, pkg.Name)
			if err != nil {
				if err != db.ErrNotFound {
					return err
				}
				continue
			}

			for _, repo := range repos {
				if !contains(pkgs[i].Repositories, repo) {
					pkgs[i].Repositories = append(pkgs[i].Repositories, repo)
				}
			}
		}

		// Get scores for each package+repository combination from the
		// database
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

		// Generate any missing scores
		if ro.GenerateScores {
			// Find repositories without scores
			var repos []string
			for _, result := range results {
				if result.Repository == "" || result.Score != nil {
					continue
				}
				if contains(repos, result.Repository) {
					continue
				}
				repos = append(repos, result.Repository)
			}

			// Generate a score for each repository
			for _, repo := range repos {
				// Attempt to generate a score
				fmt.Fprintf(os.Stderr, "Generating missing score for %s...\n", repo)
				score, err := scorecard.GenerateScore(ctx, repo)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning, couldn't generate score for %s: %s\n", repo, err)
					continue
				}

				// Add the score to the results
				for i, result := range results {
					if result.Repository != repo {
						continue
					}
					results[i].Score = score
				}
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
	s, err := tallyDB.GetScores(ctx, repo)
	if err != nil {
		return nil, err
	}
	if len(s) != 1 {
		return nil, fmt.Errorf("unexpected number of scores returned from database: %d", len(s))
	}
	score = &types.Score{
		Score:  s[0].Score,
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
	rootCmd.Flags().StringVarP(&ro.Format, "format", "f", string(bom.FormatCycloneDXJSON), fmt.Sprintf("BOM format, options=%s", bom.Formats))
	rootCmd.Flags().BoolVarP(&ro.All, "all", "a", false, "print all packages, even those without a scorecard score")
	rootCmd.Flags().StringVarP(&ro.Output, "output", "o", "short", fmt.Sprintf("output format, options=%s", output.Formats))
	rootCmd.Flags().BoolVarP(&ro.GenerateScores, "generate", "g", false, "generate scores for repositories that aren't in the database. The GITHUB_TOKEN environment variable must be set.")
	rootCmd.Flags().BoolVar(&ro.CheckForUpdate, "check-for-update", true, "check for database update")
	rootCmd.Flags().StringVar(&ro.DBRef, "db-reference", "ghcr.io/jetstack/tally/db:v1", "image reference to download database from")
	rootCmd.Flags().Var(&ro.FailOn, "fail-on", "fail if a package is found with a score <= to the given value")
}
