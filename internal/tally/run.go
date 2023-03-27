package tally

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"

	"github.com/cheggaaa/pb/v3"
	"github.com/jetstack/tally/internal/scorecard"
	"github.com/jetstack/tally/internal/types"
	"golang.org/x/sync/errgroup"
)

const pbTemplate = `{{ string . "message" }} {{ bar . "[" "-" ">" "." "]"}} {{counters . }}`

// Run finds scorecard scores for the provided packages
func Run(ctx context.Context, w io.Writer, clients []scorecard.Client, pkgRepos ...*types.PackageRepositories) (*types.Report, error) {
	// If the writer is nil then just discard anything we write
	if w == nil {
		w = io.Discard
	}

	// Map repositories to packages
	repoPkgs := map[string][]types.Package{}
	for _, pkgRepo := range pkgRepos {
		// We want to include packages without a repository in the
		// results
		repos := pkgRepo.Repositories
		if len(repos) == 0 {
			repos = []types.Repository{
				{
					Name: "",
				},
			}
		}
		for _, repo := range repos {
			repoPkgs[repo.Name] = append(repoPkgs[repo.Name], pkgRepo.Package)
		}
	}

	// Map into results
	var results []types.Result
	for repoName, pkgs := range repoPkgs {
		results = append(results, types.Result{
			Repository: types.Repository{
				Name: repoName,
			},
			Packages: pkgs,
		})
	}

	bar := pb.ProgressBarTemplate(pbTemplate).Start(len(results))
	bar.SetWriter(w)
	bar.Set(pb.CleanOnFinish, true)
	defer bar.Finish()

	for _, client := range clients {
		var g errgroup.Group
		g.SetLimit(runtime.NumCPU())
		mux := sync.RWMutex{}
		for i, result := range results {
			if result.Result != nil || result.Repository.Name == "" {
				continue
			}
			i, result := i, result
			g.Go(func() error {
				mux.Lock()
				// Tweak the message displayed in the progress bar
				// depending on the type of client
				switch client.Name() {
				case scorecard.ScorecardClientName:
					bar.Set("message", fmt.Sprintf("Generating score for %q", result.Repository.Name))
				default:
					bar.Set("message", fmt.Sprintf("Finding score for %q", result.Repository.Name))
				}
				mux.Unlock()

				scorecardResult, err := client.GetResult(ctx, result.Repository.Name)
				if err != nil && !errors.Is(err, scorecard.ErrNotFound) {
					return fmt.Errorf("getting score for %s: %w", result.Repository.Name, err)
				}
				if scorecardResult == nil {
					return nil
				}

				results[i].Result = scorecardResult

				mux.Lock()
				bar.Increment()
				mux.Unlock()

				return nil
			})
		}
		if err := g.Wait(); err != nil {
			return nil, err
		}
	}

	bar.Set("message", "DONE")

	return &types.Report{
		Results: results,
	}, nil
}
