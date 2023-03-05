package db

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Manager manages a local database
type Manager interface {
	// CreateDB creates the database from the provided sources
	CreateDB(context.Context, ...Source) error

	// DB returns the managed database
	DB() (DB, error)

	// PullDB pulls the database from a remote reference. Returns true
	// if the database was updated and false if the database was already at
	// the provided version.
	PullDB(context.Context, string) (bool, error)

	// Metadata returns metadata about the current database
	Metadata() (*Metadata, error)
}

type manager struct {
	dbDir string
	w     io.Writer
}

// NewManager returns a new manager that manages a local database
func NewManager(dbDir string, output io.Writer) (Manager, error) {
	if output == nil {
		output = io.Discard
	}

	if dbDir == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			return nil, fmt.Errorf("getting user cache dir: %s", err)
		}

		dbDir = filepath.Join(cacheDir, "tally", "db")
	}

	return &manager{
		dbDir: dbDir,
		w:     output,
	}, nil
}

// DB returns the managed database
func (m *manager) DB() (DB, error) {
	tallyDB, err := NewDB(filepath.Join(m.dbDir, "tally.db"))
	if err != nil {
		return nil, fmt.Errorf("getting database: %w", err)
	}

	return tallyDB, nil
}
