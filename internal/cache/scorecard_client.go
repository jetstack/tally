package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/jetstack/tally/internal/scorecard"
	"github.com/jetstack/tally/internal/types"
)

// ScorecardClient wraps another scorecard client, caching the scores it retrieves
type ScorecardClient struct {
	ca Cache
	scorecard.Client
}

// NewScorecardClient returns a scorecard client that caches scores from another client
func NewScorecardClient(ca Cache, client scorecard.Client) scorecard.Client {
	return &ScorecardClient{
		ca:     ca,
		Client: client,
	}
}

// GetScore attempts to get the score from the cache. Failing that it will get
// the score from the wrapped client and cache it for next time.
func (c *ScorecardClient) GetScore(ctx context.Context, repository string) (*types.Score, error) {
	score, err := c.ca.GetScore(ctx, repository)
	if err == nil {
		return score, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("getting score from cache: %w", err)
	}

	score, err = c.Client.GetScore(ctx, repository)
	if err != nil {
		return nil, fmt.Errorf("getting score from wrapped client: %w", err)
	}

	if err := c.ca.PutScore(ctx, repository, score); err != nil {
		return nil, fmt.Errorf("caching score: %w", err)
	}

	return score, nil
}
