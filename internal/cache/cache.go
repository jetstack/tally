package cache

import (
	"context"
	"errors"

	"github.com/ossf/scorecard-webapp/app/generated/models"
	_ "modernc.org/sqlite"
)

// ErrNotFound is returned when a result isn't found in the cache
var ErrNotFound = errors.New("not found in cache")

// Cache caches results
type Cache interface {
	// GetResult retrieves a scorecard result from the cache
	GetResult(ctx context.Context, repository string) (*models.ScorecardResult, error)

	// PutResult inserts a score into the cache
	PutResult(ctx context.Context, repository string, result *models.ScorecardResult) error
}
