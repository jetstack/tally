package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jetstack/tally/internal/db"
	"github.com/jetstack/tally/internal/types"
)

type dbMapper struct {
	d db.RepositoryReader
}

// DBMapper returns a mapper that retrieves repositories from the tally database
func DBMapper(d db.RepositoryReader) Mapper {
	return &dbMapper{
		d: d,
	}
}

// Repositories gets repositories from the tally database
func (m *dbMapper) Repositories(ctx context.Context, pkg types.Package) ([]string, error) {
	repos, err := m.d.GetRepositories(ctx, pkg.System, pkg.Name)
	if errors.Is(err, db.ErrNotFound) {
		return []string{}, nil
	}
	if err != nil {
		return []string{}, fmt.Errorf("database mapper: %w", err)
	}

	return repos, nil
}
