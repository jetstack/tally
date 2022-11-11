package db

import (
	"context"
	"errors"
)

// ErrNotFound is a not found error
var ErrNotFound = errors.New("not found")

// DB is the tally database
type DB interface {
	DBReader
	DBWriter

	// Close the database
	Close() error

	// Initialize the database
	Initialize(context.Context) error
}

// DBReader reads from the database
type DBReader interface {
	// GetChecks retrieves check scores for a repository. Returns
	// ErrNotFound if no checks are found.
	GetChecks(context.Context, string) ([]Check, error)

	// GetRepositories returns any repositories associated with the package
	// indicated by system and name. Returns ErrNotFound if there are no matching
	// repositories.
	GetRepositories(context.Context, string, string) ([]string, error)

	// GetScore retrieves scorecard scores for a list of repositories
	GetScores(context.Context, ...string) ([]Score, error)
}

// DBWriter writes to the database
type DBWriter interface {
	// AddChecks adds scorecard check scores to the database. If there's
	// already a check in the database for the repository in question then
	// the existing check will be updated.
	AddChecks(context.Context, ...Check) error

	// AddPackages adds packages to the database. The same package+system
	// combination can have multiple repositories associated with it.
	AddPackages(context.Context, ...Package) error

	// AddScores adds scorecard scores to the database. If there's already
	// a score in the database for the repository in question, then the
	// existing score will be updated.
	AddScores(context.Context, ...Score) error
}

// Package is a package associated with a repository
type Package struct {
	System     string
	Name       string
	Repository string
}

// Score is the aggregated scorecard score for a repository
type Score struct {
	Repository string
	Score      float64
}

// Check is a repository's score for an individual scorecard check
type Check struct {
	Name       string
	Repository string
	Score      int
}
