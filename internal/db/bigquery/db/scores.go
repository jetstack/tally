package bigquery

import (
	"context"
	"fmt"
	"strings"

	"github.com/jetstack/tally/internal/db"
	"google.golang.org/api/iterator"
)

type scoreRow struct {
	Repo  scoreRepo `bigquery:"repo"`
	Score float64   `bigquery:"score"`
}

type scoreRepo struct {
	Name string `bigquery:"name"`
}

// GetScores from the dataset
func (d *database) GetScores(ctx context.Context, repos ...string) ([]db.Score, error) {
	// TODO: remove duplicates by selecting the latest row
	q := fmt.Sprintf(`
        SELECT repo, score
        FROM `+fmt.Sprintf("`%s.%s.%s`", d.dataset.ProjectID, d.dataset.DatasetID, d.scoreTable.TableID)+`
        WHERE repo.name IN ('%s')
	ORDER BY repo.name ASC;
        `, strings.Join(repos, "', '"))

	it, err := d.bq.Query(q).Read(ctx)
	if err != nil {
		return []db.Score{}, err
	}

	var scores []db.Score
	for {
		var row scoreRow
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return []db.Score{}, err
		}

		scores = append(scores, db.Score{
			Repository: row.Repo.Name,
			Score:      row.Score,
		})
	}

	if len(scores) == 0 {
		return []db.Score{}, db.ErrNotFound
	}

	return scores, nil
}

// AddScores to the dataset
func (d *database) AddScores(ctx context.Context, scores ...db.Score) error {
	var r []*scoreRow

	for _, score := range scores {
		r = append(r, &scoreRow{
			Repo: scoreRepo{
				Name: score.Repository,
			},
			Score: score.Score,
		})
	}

	return d.scoreTable.Inserter().Put(ctx, r)
}
