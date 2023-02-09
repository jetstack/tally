package scorecard

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jetstack/tally/internal/types"
	"github.com/ossf/scorecard/v4/checker"
	"github.com/ossf/scorecard/v4/checks"
	docs "github.com/ossf/scorecard/v4/docs/checks"
	"github.com/ossf/scorecard/v4/pkg"
)

var (
	// ErrNotFound is returned by the client when it can't find a score for the
	// repository
	ErrNotFound = errors.New("score not found")

	// ErrUnexpectedResponse is returned when a scorecard client gets an unexpected
	// response from its upstream source
	ErrUnexpectedResponse = errors.New("unexpected response")

	// ErrInvalidRepository is returned when an invalid repository is
	// provided as input
	ErrInvalidRepository = errors.New("invalid repository")
)

// Client fetches scorecard results for repositories
type Client interface {
	// GetScore retrieves the scorecard score for the given repository
	GetScore(ctx context.Context, repository string) (*types.Score, error)

	// ConcurrencyLimit indicates the maximum number of concurrent invocations
	// the client supports. A value of 0 indicates that there is no limit.
	ConcurrencyLimit() int

	// Name returns the name of this client
	Name() string
}

// ScorecardClientName is the name of the scorecard client
const ScorecardClientName = "scorecard"

// ScorecardClient generates scorecard scores for repositories
type ScorecardClient struct{}

// NewScorecardClient returns a new client that generates scores itself for
// repositories
func NewScorecardClient() (Client, error) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable must be set when using the scorecard client to generate scores")
	}
	return &ScorecardClient{}, nil
}

// Name is the name of the client
func (c *ScorecardClient) Name() string {
	return ScorecardClientName
}

// GetScore generates a scorecard score with the scorecard client
func (c *ScorecardClient) GetScore(ctx context.Context, repository string) (*types.Score, error) {
	repoURI, repoClient, ossFuzzRepoClient, ciiClient, vulnsClient, err := checker.GetClients(
		ctx, repository, "", nil)
	if err != nil {
		return nil, fmt.Errorf("getting clients: %w", errors.Join(ErrNotFound, err))
	}
	defer repoClient.Close()
	if ossFuzzRepoClient != nil {
		defer ossFuzzRepoClient.Close()
	}

	enabledChecks := checks.GetAll()

	checkDocs, err := docs.Read()
	if err != nil {
		return nil, fmt.Errorf("checking docs: %s", errors.Join(ErrNotFound, err))
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
		return nil, fmt.Errorf("running scorecards: %w", errors.Join(ErrNotFound, err))
	}

	score := &types.Score{
		Checks: map[string]int{},
	}
	for _, check := range res.Checks {
		score.Checks[check.Name] = check.Score
	}

	s, err := res.GetAggregateScore(checkDocs)
	if err != nil {
		return nil, fmt.Errorf("getting aggregate score: %w", errors.Join(ErrNotFound, err))
	}
	score.Score = s

	return score, nil
}

// ConcurrencyLimit indicates that the client cannot be ran concurrently
func (c *ScorecardClient) ConcurrencyLimit() int {
	return 1
}
