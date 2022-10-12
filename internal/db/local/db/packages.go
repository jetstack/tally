package local

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jetstack/tally/internal/db"
)

// GetRepositories returns repositories associated with the provided package
func (d *database) GetRepositories(ctx context.Context, system, name string) ([]string, error) {
	q := `
        SELECT repositories.name
        FROM repositories, package_systems, packages
        WHERE packages.repository_id = repositories.id
	AND packages.system_id = package_systems.id
        AND package_systems.name IN (?)
	AND packages.name IN (?)
	ORDER BY repositories.name ASC;
        `

	rows, err := d.db.QueryContext(ctx, q, system, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []string{}, db.ErrNotFound
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
		return repos, db.ErrNotFound
	}

	return repos, nil
}

// AddPackages inserts packages into the database
func (d *database) AddPackages(ctx context.Context, pkgs ...db.Package) error {
	// Create a temporary table that stores the full name of the repository
	// along with the package
	if _, err := d.db.ExecContext(ctx, `
          CREATE TABLE packages_tmp (
            system text NOT NULL,
            name text NOT NULL,
            repository text NOT NULL,
            PRIMARY KEY (system,name,repository)
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
	        (system, name, repository)
	        VALUES 
	        `
		vals := []interface{}{}
		for _, pkg := range chunk {
			q += "(?, ?, ?),"
			vals = append(
				vals,
				pkg.System,
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

	// Populate the package_systems table with all the distinct systems
	if _, err := d.db.ExecContext(ctx, `
	  INSERT or IGNORE INTO package_systems
	  (name)
	  SELECT DISTINCT packages_tmp.system
	  FROM packages_tmp;
	`); err != nil {
		return fmt.Errorf("inserting package systems: %w", err)
	}

	// Insert the packages into the packages table, with the repository_id
	// from the repositories table and the system_id from the
	// package_systems table
	if _, err := d.db.ExecContext(ctx, `
          INSERT or IGNORE INTO packages
          (system_id, name, repository_id)
          SELECT package_systems.id, packages_tmp.name, repositories.id
          FROM packages_tmp, repositories, package_systems
          WHERE packages_tmp.repository = repositories.name AND packages_tmp.system = package_systems.name;
        `); err != nil {
		return fmt.Errorf("inserting packages from temporary table: %w", err)
	}

	// Drop the temporary packages table
	if _, err := d.db.ExecContext(ctx, `DROP TABLE packages_tmp;`); err != nil {
		return fmt.Errorf("dropping temporary packages table: %w", err)
	}

	return nil
}
