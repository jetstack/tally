package local

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jetstack/tally/internal/db"
)

// GetChecks retrieves checks from the database for a repository
func (d *database) GetChecks(ctx context.Context, repo string) ([]db.Check, error) {
	q := `
        SELECT checks.name, repositories.name, check_scores.score
        FROM checks, repositories, check_scores
        WHERE check_scores.check_id = checks.id
        AND check_scores.repository_id = repositories.id
        AND repositories.name IN (?)
	ORDER BY checks.name ASC;
        `

	rows, err := d.db.QueryContext(ctx, q, repo)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []db.Check{}, db.ErrNotFound
		}
		return []db.Check{}, fmt.Errorf("querying database: %w", err)
	}
	defer rows.Close()

	var checks []db.Check
	for rows.Next() {
		var check db.Check
		if err := rows.Scan(
			&check.Name,
			&check.Repository,
			&check.Score,
		); err != nil {
			return []db.Check{}, fmt.Errorf("scanning row: %w", err)
		}

		checks = append(checks, check)
	}

	if len(checks) == 0 {
		return checks, db.ErrNotFound
	}

	return checks, nil
}

// AddChecks inserts checks into the database
func (d *database) AddChecks(ctx context.Context, checks ...db.Check) error {
	// Create a temporary table that stores the full name of the repository
	// along with the score
	if _, err := d.db.ExecContext(ctx, `
          CREATE TABLE checks_tmp (
            name text NOT NULL,
            repository text NOT NULL,
            score integer NOT NULL,
            PRIMARY KEY (name, repository)
          ) WITHOUT ROWID;
        `); err != nil {
		return fmt.Errorf("creating temporary table: %w", err)
	}

	// Insert scores into the temporary table. Split into chunks to account
	// for the maximum number of host parameters allowed by sqlite (32766)
	for _, chunk := range chunkSlice(checks, 32766/3) {
		q := `
		INSERT INTO checks_tmp
		(name, repository, score)
		VALUES
		`
		vals := []interface{}{}
		for _, check := range chunk {
			q += "(?, ?, ?),"
			vals = append(
				vals,
				check.Name,
				check.Repository,
				check.Score,
			)
		}
		q = strings.TrimSuffix(q, ",")
		q += " ON CONFLICT(name, repository) DO UPDATE SET score=excluded.score"

		if _, err := d.db.ExecContext(ctx, q, vals...); err != nil {
			return fmt.Errorf("inserting checks: %w", err)
		}
	}

	// Populate the checks table with all the distinct types of check
	if _, err := d.db.ExecContext(ctx, `
	  INSERT or IGNORE INTO checks
	  (name)
	  SELECT DISTINCT checks_tmp.name
	  FROM checks_tmp;
        `); err != nil {
		return fmt.Errorf("executing statement: %w", err)
	}

	// Populate the repositories table with all the repositories
	if _, err := d.db.ExecContext(ctx, `
	  INSERT or IGNORE INTO repositories
	  (name)
	  SELECT DISTINCT checks_tmp.repository
	  FROM checks_tmp;
	`); err != nil {
		return fmt.Errorf("inserting repositories: %w", err)
	}

	// Insert the checks into the check_scores table, with the repository id
	// from the repositories table and the check id from the checks table
	if _, err := d.db.ExecContext(ctx, `
          INSERT INTO check_scores
          (check_id, repository_id, score)
          SELECT checks.id, repositories.id, checks_tmp.score
          FROM repositories, checks, checks_tmp
          WHERE checks_tmp.repository = repositories.name AND checks_tmp.name = checks.name
	  ON CONFLICT(check_id, repository_id) DO UPDATE SET score=excluded.score;
        `); err != nil {
		return fmt.Errorf("executing statement: %w", err)
	}

	// Drop the temporary checks table
	if _, err := d.db.ExecContext(ctx, `DROP TABLE checks_tmp;`); err != nil {
		return fmt.Errorf("executing statement: %w", err)
	}

	return nil
}
