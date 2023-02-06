package scorecard

import (
	"context"
	"fmt"

	"github.com/jetstack/tally/internal/types"
	"github.com/ossf/scorecard/v4/checker"
	"github.com/ossf/scorecard/v4/checks"
	docs "github.com/ossf/scorecard/v4/docs/checks"
	"github.com/ossf/scorecard/v4/pkg"
)

// GenerateScore generates a scorecard score for the provided repository
func GenerateScore(ctx context.Context, repository string) (*types.Score, error) {
	repoURI, repoClient, ossFuzzRepoClient, ciiClient, vulnsClient, err := checker.GetClients(
		ctx, repository, "", nil)
	if err != nil {
		return nil, fmt.Errorf("getting clients: %w", err)
	}
	defer repoClient.Close()
	if ossFuzzRepoClient != nil {
		defer ossFuzzRepoClient.Close()
	}

	enabledChecks := checks.GetAll()

	checkDocs, err := docs.Read()
	if err != nil {
		return nil, fmt.Errorf("checking docs: %s", err)
	}

	res, err := pkg.RunScorecard(
		ctx,
		repoURI,
		"HEAD",
		0,
		enabledChecks,
		repoClient,
		ossFuzzRepoClient,
		ciiClient,
		vulnsClient,
	)
	if err != nil {
		return nil, fmt.Errorf("running scorecards: %w", err)
	}

	score := &types.Score{
		Checks: map[string]int{},
	}
	for _, check := range res.Checks {
		score.Checks[check.Name] = check.Score
	}

	s, err := res.GetAggregateScore(checkDocs)
	if err != nil {
		return nil, fmt.Errorf("getting aggregate score: %w", err)
	}
	score.Score = s

	return score, nil
}
