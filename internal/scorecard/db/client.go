package scorecard

import (
	"context"
	"errors"
	"fmt"

	"github.com/jetstack/tally/internal/db"
	"github.com/jetstack/tally/internal/scorecard"
	"github.com/ossf/scorecard-webapp/app/generated/models"
)

// ClientName is the name of the client
const ClientName = "tally-db"

// Client fetches scores from the tally database
type Client struct {
	d db.ScoreReader
}

// NewClient returns a client that gets scores from the tally database
func NewClient(d db.ScoreReader) scorecard.Client {
	return &Client{d}
}

// Name returns the name of the client
func (c *Client) Name() string {
	return ClientName
}

// GetScore retrieves scores from the tally database
func (c *Client) GetResult(ctx context.Context, repository string) (*models.ScorecardResult, error) {
	// Get the aggregate score
	s, err := c.d.GetScores(ctx, repository)
	if errors.Is(err, db.ErrNotFound) {
		return nil, scorecard.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(s) != 1 {
		return nil, fmt.Errorf("expected 1 score but got %d: %w", len(s), scorecard.ErrUnexpectedResponse)
	}
	result := &models.ScorecardResult{
		Repo: &models.Repo{
			Name: repository,
		},
		Score: s[0].Score,
	}

	// Get the individual check scores
	checks, err := c.d.GetChecks(ctx, repository)
	if errors.Is(err, db.ErrNotFound) {
		return nil, scorecard.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	for _, check := range checks {
		result.Checks = append(result.Checks, &models.ScorecardCheck{
			Name:  check.Name,
			Score: int64(check.Score),
		})
	}

	return result, nil
}

// ConcurrencyLimit indicates that the client cannot be ran concurrently
func (c *Client) ConcurrencyLimit() int {
	return 1
}
