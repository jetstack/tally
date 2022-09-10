package bigquery

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/jetstack/tally/internal/db"
	"google.golang.org/api/iterator"
)

type scoreSrc struct {
	db        db.ScoreWriter
	read      bqRead
	batchSize int
}

type scoreRow struct {
	Repository string  `bigquery:"repository"`
	Score      float64 `bigquery:"score"`
}

// NewScoreSource returns a new source that fetches scores from the scorecard
// dataset
func NewScoreSource(bq *bigquery.Client, db db.DB) db.Source {
	return &scoreSrc{
		db: db,
		read: func(ctx context.Context, qs string) (bqRowIterator, error) {
			return bq.Query(qs).Read(ctx)
		},
		batchSize: 500000,
	}
}

// String returns the name of the source.
func (s *scoreSrc) String() string {
	return "scorecard scores"
}

// Update fetches scores from the scorecard dataset and adds them to the
// database
func (s *scoreSrc) Update(ctx context.Context) error {
	qs := `
	SELECT DISTINCT scorecard.repo.name as repository, MIN(scorecard.score) as score
        FROM ` + "`openssf.scorecardcron.scorecard-v2_latest`" + ` scorecard
	WHERE scorecard.score >= 0
        GROUP BY scorecard.repo.name
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
		var scores []db.Score

		// Insert after each batch of rows to avoid storing the
		// entire dataset in memory
		for len(scores) < s.batchSize {
			var row scoreRow
			err := it.Next(&row)
			if err == iterator.Done {
				done = true
				break
			}
			if err != nil {
				return fmt.Errorf("calling Next on iterator: %w", err)
			}
			i++

			scores = append(scores, db.Score{
				Repository: row.Repository,
				Score:      row.Score,
			})
		}

		if err := s.db.AddScores(ctx, scores...); err != nil {
			return fmt.Errorf("adding scores: %w", err)
		}

		scores = nil
	}
	return nil
}
