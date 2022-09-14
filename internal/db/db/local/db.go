package local

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jetstack/tally/internal/db"

	_ "github.com/mattn/go-sqlite3"
)

// Option is a functional option that configures the database
type Option func(db *database)

// WithVacuumOnClose vacuums the database when it's closed
func WithVacuumOnClose() Option {
	return func(db *database) {
		db.vacuum = true
	}
}

type database struct {
	db     *sql.DB
	vacuum bool
}

// NewDB returns a new database at the provided path
func NewDB(dbPath string, opts ...Option) (db.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	d := &database{
		db: db,
	}

	for _, o := range opts {
		o(d)
	}

	return d, nil
}

// Close closes the database
func (d *database) Close() error {
	if d.vacuum {
		if _, err := d.db.Exec(`VACUUM;`); err != nil {
			return fmt.Errorf("running vacuum: %w", err)
		}

	}
	return d.db.Close()
}

// Intialize sets up the database
func (d *database) Initialize(ctx context.Context) error {
	if _, err := d.db.Exec(`
	PRAGMA foreign_keys=ON;

	CREATE TABLE repositories (
	  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	  name text NOT NULL UNIQUE
	);

	CREATE TABLE package_systems (
	  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	  name text NOT NULL UNIQUE
	);

	CREATE TABLE packages (
	  name text NOT NULL,
	  system_id interger NOT NULL,
	  repository_id integer NOT NULL,
	  FOREIGN KEY (system_id) REFERENCES package_systems (id),
	  FOREIGN KEY (repository_id) REFERENCES repositories (id),
	  PRIMARY KEY (name, system_id, repository_id)
	) WITHOUT ROWID;

	CREATE TABLE scores (
	  score real NOT NULL,
	  repository_id integer NOT NULL,
	  FOREIGN KEY (repository_id) REFERENCES repositories (id),
	  PRIMARY KEY (repository_id)
	) WITHOUT ROWID;

	CREATE TABLE checks (
	  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	  name text NOT NULL UNIQUE
	);

	CREATE TABLE check_scores (
	  check_id integer NOT NULL,
	  repository_id integer NOT NULL,
	  score integer NOT NULL,
	  FOREIGN KEY (check_id) REFERENCES checks (id),
	  FOREIGN KEY (repository_id) REFERENCES repositories (id),
	  PRIMARY KEY (check_id, repository_id)
	) WITHOUT ROWID;

	`); err != nil {
		return fmt.Errorf("initializing database: %w", err)
	}

	return nil
}
