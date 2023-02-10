package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/jetstack/tally/internal/scorecard"
	"github.com/ossf/scorecard-webapp/app/generated/models"
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

// GetResult attempts to get the scorecard result from the cache. Failing that it will get
// the scorecard result from the wrapped client and cache it for next time.
func (c *ScorecardClient) GetResult(ctx context.Context, repository string) (*models.ScorecardResult, error) {
	result, err := c.ca.GetResult(ctx, repository)
	if err == nil {
		return result, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("getting scorecard result from cache: %w", err)
	}

	result, err = c.Client.GetResult(ctx, repository)
	if err != nil {
		return nil, fmt.Errorf("getting scorecard result from wrapped client: %w", err)
	}

	if err := c.ca.PutResult(ctx, repository, result); err != nil {
		return nil, fmt.Errorf("caching scorecard result: %w", err)
	}

	return result, nil
}
