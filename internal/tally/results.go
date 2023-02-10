package tally

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"

	"github.com/cheggaaa/pb/v3"
	"github.com/jetstack/tally/internal/repositories"
	"github.com/jetstack/tally/internal/scorecard"
	"github.com/jetstack/tally/internal/types"
	"golang.org/x/sync/errgroup"
)

const pbTemplate = `{{ string . "message" }} {{ bar . "[" "-" ">" "." "]"}} {{counters . }}`

// Results finds scorecard scores for the provided packages
func Results(ctx context.Context, w io.Writer, repoMapper repositories.Mapper, clients []scorecard.Client, pkgs ...types.Package) ([]types.Result, error) {
	// If the writer is nil then just discard anything we write
	if w == nil {
		w = io.Discard
	}

	// Map repositories to packages
	repoPkgs := map[string][]types.Package{}
	for _, pkg := range pkgs {
		repos, err := repoMapper.Repositories(ctx, pkg)
		if err != nil {
			return nil, fmt.Errorf("getting repositories for package %s/%s: %w", pkg.System, pkg.Name, err)
		}
		// We want to include packages without a repository in the
		// results
		if len(repos) == 0 {
			repos = []string{""}
		}
		for _, repo := range repos {
			repoPkgs[repo] = append(repoPkgs[repo], pkg)
		}
	}

	// Map into results
	var results []types.Result
	for repo, pkgs := range repoPkgs {
		results = append(results, types.Result{
			Repository: repo,
			Packages:   pkgs,
		})
	}

	bar := pb.ProgressBarTemplate(pbTemplate).Start(len(results))
	bar.SetWriter(w)
	bar.Set(pb.CleanOnFinish, true)
	defer bar.Finish()

	for _, client := range clients {
		limit := client.ConcurrencyLimit()
		if limit < 1 {
			limit = runtime.NumCPU()
		}
		var g errgroup.Group
		g.SetLimit(limit)
		mux := sync.RWMutex{}
		for i, result := range results {
			if result.ScorecardResult != nil || result.Repository == "" {
				continue
			}
			i, result := i, result
			g.Go(func() error {
				mux.Lock()
				// Tweak the message displayed in the progress bar
				// depending on the type of client
				switch client.Name() {
				case scorecard.ScorecardClientName:
					bar.Set("message", fmt.Sprintf("Generating score for %q", result.Repository))
				default:
					bar.Set("message", fmt.Sprintf("Finding score for %q", result.Repository))
				}
				mux.Unlock()

				scorecardResult, err := client.GetResult(ctx, result.Repository)
				if err != nil && !errors.Is(err, scorecard.ErrNotFound) {
					return fmt.Errorf("getting score for %s: %w", result.Repository, err)
				}
				if scorecardResult == nil {
					return nil
				}

				results[i].ScorecardResult = scorecardResult

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

	return results, nil
}
