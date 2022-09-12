package tally

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"github.com/jetstack/tally/internal/scorecard"
	"github.com/jetstack/tally/internal/types"
	"github.com/ossf/scorecard/v4/checker"
	"github.com/ossf/scorecard/v4/checks"
	docs "github.com/ossf/scorecard/v4/docs/checks"
	"github.com/ossf/scorecard/v4/pkg"
)

// AddScoresFromScorecardTable encriches the results with scores from
// the provided table.
func AddScoresFromScorecardTable(ctx context.Context, table scorecard.Table, results []types.Result) ([]types.Result, error) {
	var repositories []string
	for _, result := range results {
		if result.Repository == "" || result.Score != nil {
			continue
		}
		repositories = append(repositories, result.Repository)
	}

	rows, err := table.SelectWhereRepositoryIn(ctx, repositories)
	if err != nil {
		return nil, err
	}

	// Add scores to the list of packages
	for _, row := range rows {
		for i, result := range results {
			if result.Repository == row.Repo.Name {
				results[i].Score = &types.Score{
					Score: row.Score,
				}
			}
		}
	}

	return results, nil
}

// GenerateScores generates scores for the packages that have a repository
// value but no score.
func GenerateScores(ctx context.Context, table scorecard.Table, results []types.Result) ([]types.Result, error) {
	repoScores := map[string]*scorecard.Row{}
	for i, result := range results {
		if !(strings.HasPrefix(result.Repository, "github.com/") && result.Score == nil) {
			continue
		}

		date := civil.DateOf(time.Now())
		row, ok := repoScores[result.Repository]
		if !ok {
			fmt.Fprintf(os.Stderr, "Generating score for %s...\n", result.Repository)
			score, err := generateScoreForRepository(ctx, result.Repository)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error generating score for %s: %s\n", result.Repository, err)
				continue
			}

			row = &scorecard.Row{
				Repo: scorecard.Repo{
					Name: result.Repository,
				},
				Score: score,
				Date:  date,
			}

			if table != nil {
				if err := table.Insert(ctx, row); err != nil {
					return nil, err
				}
			}

			repoScores[result.Repository] = row
		}

		results[i].Score = &types.Score{
			Score: row.Score,
		}
	}

	return results, nil
}

func generateScoreForRepository(ctx context.Context, repository string) (float64, error) {
	repoURI, repoClient, ossFuzzRepoClient, ciiClient, vulnsClient, err := checker.GetClients(
		ctx, repository, "", nil)
	if err != nil {
		return 0, fmt.Errorf("getting clients: %w", err)
	}
	defer repoClient.Close()
	if ossFuzzRepoClient != nil {
		defer ossFuzzRepoClient.Close()
	}

	enabledChecks := checks.GetAll()

	checkDocs, err := docs.Read()
	if err != nil {
		return 0, fmt.Errorf("checking docs: %s", err)
	}

	res, err := pkg.RunScorecards(
		ctx,
		repoURI,
		"HEAD",
		enabledChecks,
		repoClient,
		ossFuzzRepoClient,
		ciiClient,
		vulnsClient,
	)
	if err != nil {
		return 0, fmt.Errorf("running scorecards: %w", err)
	}

	return res.GetAggregateScore(checkDocs)
}
