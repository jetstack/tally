package cache

import (
	"context"
	"errors"

	"github.com/jetstack/tally/internal/types"
	_ "modernc.org/sqlite"
)

// ErrNotFound is returned when a result isn't found in the cache
var ErrNotFound = errors.New("not found in cache")

// Cache caches results
type Cache interface {
	// GetScore retrieves a score from the cache
	GetScore(ctx context.Context, repository string) (*types.Score, error)

	// PutScore inserts a score into the cache
	PutScore(ctx context.Context, repository string, score *types.Score) error
}
