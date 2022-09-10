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

	// Initialize will setup the database. Returns an error if the store has
	// already been initialized.
	Initialize(context.Context) error
}

// DBReader reads data from the tally database
type DBReader interface {
	PackageReader
	ScoreReader
	CheckReader
}

// DBWriter writes data to the tally database
type DBWriter interface {
	PackageWriter
	ScoreWriter
	CheckWriter
}

// Package is a package associated with a repository
type Package struct {
	System     string
	Name       string
	Repository string
}

// PackageWriter reads package information from the database
type PackageReader interface {
	// GetRepositories returns any repositories associated with the package
	// indicated by system and name. Returns ErrNotFound if there are no matching
	// repositories.
	GetRepositories(context.Context, string, string) ([]string, error)
}

// PackageWriter writes packages to the database
type PackageWriter interface {
	// AddPackages adds packages to the database. The same package+system
	// combination can have multiple repositories associated with it.
	AddPackages(context.Context, ...Package) error
}

// Score is the aggregated scorecard score for a repository
type Score struct {
	Repository string
	Score      float64
}

// ScoreReader reads scores from the database
type ScoreReader interface {
	// GetScore retrieves the overall scorecard score for a repository
	GetScore(context.Context, string) (float64, error)
}

// ScoreWriter writes scores to the database
type ScoreWriter interface {
	// AddScores adds scorecard scores to the database. If there's already
	// a score in the database for the repository in question, then the
	// existing score will be updated.
	AddScores(context.Context, ...Score) error
}

// Check is a repository's score for an individual scorecard check
type Check struct {
	Name       string
	Repository string
	Score      int
}

// CheckReader reads checks from the database
type CheckReader interface {
	// GetChecks retrieves check scores for a repository. Returns
	// ErrNotFound if no checks are found.
	GetChecks(context.Context, string) ([]Check, error)
}

// CheckWriter writes checks to the database
type CheckWriter interface {
	// AddChecks adds scorecard check scores to the database. If there's
	// already a check in the database for the repository in question then
	// the existing check will be updated.
	AddChecks(context.Context, ...Check) error
}
