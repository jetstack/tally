package local

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jetstack/tally/internal/db"
)

// GetScore retrieves a repository's score from the database
func (d *database) GetScore(ctx context.Context, repo string) (float64, error) {
	q := `
        SELECT scores.score
        FROM repositories, scores
        WHERE scores.repository_id = repositories.id
        AND repositories.name IN (?)
        `

	var score float64
	err := d.db.QueryRowContext(ctx, q, repo).Scan(&score)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return 0, db.ErrNotFound
	case err != nil:
		return 0, err
	default:
		return score, nil
	}
}

// AddScores inserts scores into the database
func (d *database) AddScores(ctx context.Context, scores ...db.Score) error {
	// Create a temporary table that stores the full name of the repository
	// along with the score
	if _, err := d.db.ExecContext(ctx, `
          CREATE TABLE scores_tmp (
            repository text NOT NULL,
	    score real NOT NULL,
            PRIMARY KEY (repository)
          ) WITHOUT ROWID;
        `); err != nil {
		return fmt.Errorf("creating temporary table: %w", err)
	}

	// Insert scores into the temporary table. Split into chunks to account
	// for the maximum number of host parameters allowed by sqlite (32766)
	for _, chunk := range chunkSlice(scores, 32766/2) {
		q := `
	        INSERT INTO scores_tmp
	        (repository, score)
	        VALUES 
	        `
		vals := []interface{}{}
		for _, score := range chunk {
			q += "(?, ?),"
			vals = append(
				vals,
				score.Repository,
				score.Score,
			)
		}
		q = strings.TrimSuffix(q, ",")
		q += " ON CONFLICT(repository) DO UPDATE SET score=excluded.score"

		if _, err := d.db.ExecContext(ctx, q, vals...); err != nil {
			return fmt.Errorf("inserting scores: %w", err)
		}
	}

	// Populate the repositories table with all the repositories
	if _, err := d.db.ExecContext(ctx, `
	  INSERT or IGNORE INTO repositories
	  (name)
	  SELECT DISTINCT scores_tmp.repository
	  FROM scores_tmp;
	`); err != nil {
		return fmt.Errorf("inserting repositories: %w", err)
	}

	// Insert the scores into the scores table, with the repository_id
	// from the repositories table
	if _, err := d.db.ExecContext(ctx, `
          INSERT INTO scores
          (repository_id, score)
          SELECT repositories.id, scores_tmp.score
          FROM repositories, scores_tmp
          WHERE scores_tmp.repository = repositories.name
	  ON CONFLICT(repository_id) DO UPDATE SET score=excluded.score;
        `); err != nil {
		return fmt.Errorf("inserting scores from temporary table: %w", err)
	}

	// Drop the temporary scores table
	if _, err := d.db.ExecContext(ctx, `DROP TABLE scores_tmp;`); err != nil {
		return fmt.Errorf("dropping temporary scores table: %w", err)
	}

	return nil
}
