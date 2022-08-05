package tally

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"github.com/ossf/scorecard/v4/checker"
	"github.com/ossf/scorecard/v4/checks"
	docs "github.com/ossf/scorecard/v4/docs/checks"
	"github.com/ossf/scorecard/v4/pkg"
	"github.com/ribbybibby/tally/internal/depsdev"
	"github.com/ribbybibby/tally/internal/scorecard"
)

// Result is the result of tally for a particular package
type Result struct {
	Package    *Package `json:"package"`
	Repository string   `json:"repository,omitempty"`
	Score      float64  `json:"score,omitempty"`
	Date       string   `json:"date,omitempty"`
	Table      string   `json:"table,omitempty"`
}

// AddRepositoriesFromDepsDev encriches the results with repository
// information taken from the deps.dev dataset.
func AddRepositoriesFromDepsDev(ctx context.Context, bq *bigquery.Client, results []Result) ([]Result, error) {
	t := depsdev.NewPackageVersionToProjectTable(bq)

	var pkgv []depsdev.PackageVersion
	for _, result := range results {
		if result.Package == nil || result.Repository != "" || result.Package.Type.DepsDevSystem() == "" {
			continue
		}
		pkgv = append(pkgv, depsdev.PackageVersion{
			Name:    result.Package.Name,
			Version: result.Package.Version,
			System:  result.Package.Type.DepsDevSystem(),
		})
	}

	rows, err := t.SelectGithubProjectsWherePackageVersions(ctx, pkgv)
	if err != nil {
		return nil, err
	}

	// Include the packages that don't have a repository
	for i, result := range results {
		for _, row := range rows {
			if result.Package.Type.DepsDevSystem() == row.System && result.Package.Name == row.Name && result.Package.Version == row.Version {
				results[i].Repository = "github.com/" + row.ProjectName
			}
		}
	}

	return results, nil
}

// AddScoresFromScorecardLatest encriches the results with scores from
// the latest scorecard dataset.
func AddScoresFromScorecardLatest(ctx context.Context, bq *bigquery.Client, results []Result) ([]Result, error) {
	table, err := scorecard.NewTable(bq, "openssf.scorecardcron.scorecard-v2_latest")
	if err != nil {
		return nil, err
	}

	return AddScoresFromScorecardTable(ctx, table, results)
}

// AddScoresFromScorecardTable encriches the results with scores from
// the provided table.
func AddScoresFromScorecardTable(ctx context.Context, table scorecard.Table, results []Result) ([]Result, error) {
	var repositories []string
	for _, result := range results {
		if result.Repository == "" || result.Score > 0 {
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
				results[i].Score = row.Score
				results[i].Date = row.Date.String()
				results[i].Table = table.String()
			}
		}
	}

	return results, nil
}

// GenerateScores generates scores for the packages that have a repository
// value but no score.
func GenerateScores(ctx context.Context, table scorecard.Table, results []Result) ([]Result, error) {
	repoScores := map[string]*scorecard.Row{}
	for i, result := range results {
		if !(strings.HasPrefix(result.Repository, "github.com/") && result.Score == 0) {
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

		results[i].Score = row.Score
		results[i].Date = row.Date.String()
		results[i].Table = table.String()
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
