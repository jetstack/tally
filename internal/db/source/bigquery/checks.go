package bigquery

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/jetstack/tally/internal/db"
	"google.golang.org/api/iterator"
)

type checkSrc struct {
	db        db.CheckWriter
	read      bqRead
	batchSize int
}

type checkRow struct {
	Repository string `bigquery:"repository"`
	Name       string `bigquery:"name"`
	Score      int    `bigquery:"score"`
}

// NewCheckSource returns a source that fetches scorecard check scores from the
// scorecard dataset
func NewCheckSource(bq *bigquery.Client, db db.DB) db.Source {
	return &checkSrc{
		db: db,
		read: func(ctx context.Context, qs string) (bqRowIterator, error) {
			return bq.Query(qs).Read(ctx)
		},
		batchSize: 500000,
	}
}

// String returns the name of the source.
func (s *checkSrc) String() string {
	return "scorecard checks"
}

// Update fetches scores from the scorecard dataset and adds them to the
// database
func (s *checkSrc) Update(ctx context.Context) error {
	qs := `
        SELECT DISTINCT scorecard.repo.name as repository, checks.name, checks.score
        FROM ` + "`openssf.scorecardcron.scorecard-v2_latest`" + ` scorecard
        CROSS JOIN UNNEST(scorecard.checks) as checks
        WHERE checks.score > 0;
	`
	it, err := s.read(ctx, qs)
	if err != nil {
		return err
	}

	var (
		i    int
		done bool
	)
	for !done {
		var checks []db.Check

		// Insert after each batch of rows to avoid storing the
		// entire dataset in memory
		for len(checks) < s.batchSize {
			var row checkRow
			err := it.Next(&row)
			if err == iterator.Done {
				done = true
				break
			}
			if err != nil {
				return fmt.Errorf("calling Next on iterator: %w", err)
			}
			i++

			checks = append(checks, db.Check{
				Repository: row.Repository,
				Name:       row.Name,
				Score:      row.Score,
			})
		}

		if err := s.db.AddChecks(ctx, checks...); err != nil {
			return fmt.Errorf("adding checks: %w", err)
		}

		checks = nil
	}
	return nil
}
