package repositories

import (
	"context"
	"fmt"

	"github.com/jetstack/tally/internal/types"
)

// Mapper maps a package to associated repositories
type Mapper interface {
	// Repositories returns repositories for the given package.
	Repositories(ctx context.Context, pkg types.Package) ([]string, error)
}

type mapper struct {
	mappers []Mapper
}

// From returns a mapper that attempts to retrieve repositories from
// multiple mappers
func From(mappers ...Mapper) Mapper {
	return &mapper{
		mappers: mappers,
	}
}

// GetRepositories iterates through the mappers and returns all the repositories
func (m *mapper) Repositories(ctx context.Context, pkg types.Package) ([]string, error) {
	var repositories []string
	for _, mpr := range m.mappers {
		repos, err := mpr.Repositories(ctx, pkg)
		if err != nil {
			return []string{}, fmt.Errorf("getting repositories: %w", err)
		}
		for _, repo := range repos {
			if contains(repositories, repo) {
				continue
			}

			repositories = append(repositories, repo)
		}
	}

	return repositories, nil
}

func contains(vals []string, val string) bool {
	for _, v := range vals {
		if v == val {
			return true
		}
	}

	return false
}
