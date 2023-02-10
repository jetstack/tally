package scorecard

import (
	"context"
	"errors"
	"fmt"

	"github.com/jetstack/tally/internal/db"
	"github.com/jetstack/tally/internal/scorecard"
	"github.com/jetstack/tally/internal/types"
)

// Client fetches scores from the tally database
type Client struct {
	d db.ScoreReader
}

// NewClient returns a client that gets scores from the tally database
func NewClient(d db.ScoreReader) scorecard.Client {
	return &Client{d}
}

// GetScore retrieves scores from the tally database
func (c *Client) GetScore(ctx context.Context, repository string) (*types.Score, error) {
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
	score := &types.Score{
		Score:  s[0].Score,
		Checks: map[string]int{},
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
		score.Checks[check.Name] = check.Score
	}

	return score, nil
}

// ConcurrencyLimit indicates that the client cannot be ran concurrently
func (c *Client) ConcurrencyLimit() int {
	return 1
}
