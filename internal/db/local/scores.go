package local

import (
	"context"
	"fmt"
	"strings"

	"github.com/jetstack/tally/internal/db"
)

// GetScores retrieves scores for one or more repositories
func (d *database) GetScores(ctx context.Context, repos ...string) ([]db.Score, error) {
	vals := []interface{}{}
	placeholders := []string{}
	for _, repo := range repos {
		vals = append(vals, repo)
		placeholders = append(placeholders, "?")
	}

	q := fmt.Sprintf(`
        SELECT repositories.name, scores.score
        FROM repositories, scores
        WHERE scores.repository_id = repositories.id
        AND repositories.name IN (%s)
	ORDER BY repositories.name ASC;
        `, strings.Join(placeholders, ", "))

	rows, err := d.db.QueryContext(ctx, q, vals...)
	if err != nil {
		return []db.Score{}, fmt.Errorf("querying dataabase: %w", err)
	}
	defer rows.Close()

	var scores []db.Score
	for rows.Next() {
		var score db.Score
		if err := rows.Scan(
			&score.Repository,
			&score.Score,
		); err != nil {
			return []db.Score{}, fmt.Errorf("scanning row: %w", err)
		}

		scores = append(scores, score)
	}

	if len(scores) == 0 {
		return []db.Score{}, db.ErrNotFound
	}

	return scores, nil
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
