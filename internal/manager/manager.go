package manager

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jetstack/tally/internal/db"
	"github.com/jetstack/tally/internal/db/local"
)

// Option is a functional option that configures the manager
type Option func(mgr *manager)

// WithDir is a functional option that configures the local database directory
func WithDir(dbDir string) Option {
	return func(mgr *manager) {
		mgr.dbDir = dbDir
	}
}

// WithWriter is a functional option that configures an io.Writer that the
// manager will write output and progress bars to
func WithWriter(w io.Writer) Option {
	return func(mgr *manager) {
		mgr.w = w
	}
}

// Manager manages a local database
type Manager interface {
	// CreateDB creates the database from the provided sources
	CreateDB(context.Context, ...db.Source) error

	// DB returns the managed database
	DB() (db.DB, error)

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
func NewManager(opts ...Option) (Manager, error) {
	mgr := &manager{
		w: io.Discard,
	}
	for _, opt := range opts {
		opt(mgr)
	}

	if mgr.dbDir == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			return nil, fmt.Errorf("getting user cache dir: %s", err)
		}

		mgr.dbDir = filepath.Join(cacheDir, "tally", "db")
	}

	return mgr, nil
}

// DB returns the managed database
func (m *manager) DB() (db.DB, error) {
	tallyDB, err := local.NewDB(filepath.Join(m.dbDir, "tally.db"))
	if err != nil {
		return nil, fmt.Errorf("getting database: %w", err)
	}

	return tallyDB, nil
}
