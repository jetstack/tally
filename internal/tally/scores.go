package tally

import (
	"context"
	"fmt"

	"github.com/ossf/scorecard/v4/checker"
	"github.com/ossf/scorecard/v4/checks"
	docs "github.com/ossf/scorecard/v4/docs/checks"
	"github.com/ossf/scorecard/v4/pkg"
)

func GenerateScore(ctx context.Context, repository string) (float64, error) {
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
