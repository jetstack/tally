package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// ErrNotFound is a not found error
var ErrNotFound = errors.New("not found")

// DB is the tally database
type DB interface {
	Reader
	Writer

	// Close the database
	Close() error

	// Initialize the database
	Initialize(context.Context) error
}

// Reader reads from the database
type Reader interface {
	GetRepositories(context.Context, string, string) ([]string, error)
}

// Writer writes to the database
type Writer interface {
	// AddPackages adds packages to the database. The same package+system
	// combination can have multiple repositories associated with it.
	AddPackages(context.Context, ...Package) error
}

// Package is a package associated with a repository
type Package struct {
	Type       string
	Name       string
	Repository string
}

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
func NewDB(dbPath string, opts ...Option) (DB, error) {
	db, err := sql.Open("sqlite", dbPath)
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

	CREATE TABLE package_types (
	  id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	  name text NOT NULL UNIQUE
	);

	CREATE TABLE packages (
	  name text NOT NULL,
	  type_id interger NOT NULL,
	  repository_id integer NOT NULL,
	  FOREIGN KEY (type_id) REFERENCES package_types (id),
	  FOREIGN KEY (repository_id) REFERENCES repositories (id),
	  PRIMARY KEY (name, type_id, repository_id)
	) WITHOUT ROWID;
	`); err != nil {
		return fmt.Errorf("initializing database: %w", err)
	}

	return nil
}

// AddPackages inserts packages into the database
func (d *database) AddPackages(ctx context.Context, pkgs ...Package) error {
	// Create a temporary table that stores the full name of the repository
	// along with the package
	if _, err := d.db.ExecContext(ctx, `
          CREATE TABLE packages_tmp (
            type text NOT NULL,
            name text NOT NULL,
            repository text NOT NULL,
            PRIMARY KEY (type,name,repository)
          ) WITHOUT ROWID;
        `); err != nil {
		return fmt.Errorf("creating temporary table: %w", err)
	}

	// Insert packages into the temporary table. Split into chunks to
	// account for the maximum number of host parameters allowed by sqlite
	// (32766).
	for _, chunk := range chunkSlice(pkgs, 32766/3) {
		q := `
	        INSERT or IGNORE INTO packages_tmp
	        (type, name, repository)
	        VALUES 
	        `
		vals := []interface{}{}
		for _, pkg := range chunk {
			q += "(?, ?, ?),"
			vals = append(
				vals,
				pkg.Type,
				pkg.Name,
				pkg.Repository,
			)
		}
		q = strings.TrimSuffix(q, ",")

		if _, err := d.db.ExecContext(ctx, q, vals...); err != nil {
			return fmt.Errorf("inserting packages: %w", err)
		}
	}

	// Populate the repositories table with all the repositories
	if _, err := d.db.ExecContext(ctx, `
	  INSERT or IGNORE INTO repositories
	  (name)
	  SELECT DISTINCT packages_tmp.repository
	  FROM packages_tmp;
	`); err != nil {
		return fmt.Errorf("inserting repositories: %w", err)
	}

	// Populate the package_types table with all the distinct systems
	if _, err := d.db.ExecContext(ctx, `
	  INSERT or IGNORE INTO package_types
	  (name)
	  SELECT DISTINCT packages_tmp.type
	  FROM packages_tmp;
	`); err != nil {
		return fmt.Errorf("inserting package systems: %w", err)
	}

	// Insert the packages into the packages table, with the repository_id
	// from the repositories table and the type_id from the
	// package_types table
	if _, err := d.db.ExecContext(ctx, `
          INSERT or IGNORE INTO packages
          (type_id, name, repository_id)
          SELECT package_types.id, packages_tmp.name, repositories.id
          FROM packages_tmp, repositories, package_types
          WHERE packages_tmp.repository = repositories.name AND packages_tmp.type = package_types.name;
        `); err != nil {
		return fmt.Errorf("inserting packages from temporary table: %w", err)
	}

	// Drop the temporary packages table
	if _, err := d.db.ExecContext(ctx, `DROP TABLE packages_tmp;`); err != nil {
		return fmt.Errorf("dropping temporary packages table: %w", err)
	}

	return nil
}

// GetRepositories returns repositories associated with the provided package
func (d *database) GetRepositories(ctx context.Context, pkgType, name string) ([]string, error) {
	q := `
        SELECT repositories.name
        FROM repositories, package_types, packages
        WHERE packages.repository_id = repositories.id
	AND packages.type_id = package_types.id
        AND package_types.name IN (?)
	AND packages.name IN (?)
	ORDER BY repositories.name ASC;
        `

	rows, err := d.db.QueryContext(ctx, q, pkgType, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []string{}, ErrNotFound
		}
		return []string{}, fmt.Errorf("querying database: %w", err)
	}
	defer rows.Close()

	var repos []string
	for rows.Next() {
		var repo string
		if err := rows.Scan(&repo); err != nil {
			return []string{}, fmt.Errorf("scanning row: %w", err)
		}

		repos = append(repos, repo)
	}

	if len(repos) == 0 {
		return repos, ErrNotFound
	}

	return repos, nil
}
