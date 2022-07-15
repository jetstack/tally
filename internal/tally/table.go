package tally

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"google.golang.org/api/iterator"
)

type Table interface {
	GetScores(context.Context, *bigquery.Client, []string) ([]*scorecardResponse, error)
	InsertScore(context.Context, *bigquery.Client, string, float64, civil.Date) error
	String() string
}

type table struct {
	ProjectID string
	Dataset   string
	Table     string
}

func NewTable(projectID, tableRef string) (Table, error) {
	parts := strings.Split(tableRef, ".")
	if len(parts) == 2 {
		if projectID == "" {
			return nil, fmt.Errorf("must provide project id when absent from table reference")
		}
		return &table{
			ProjectID: projectID,
			Dataset:   parts[0],
			Table:     parts[1],
		}, nil
	}
	if len(parts) == 3 {
		return &table{
			ProjectID: parts[0],
			Dataset:   parts[1],
			Table:     parts[2],
		}, nil
	}

	return nil, fmt.Errorf("invalid table reference: %s", tableRef)
}

func (t *table) String() string {
	return strings.Join([]string{t.ProjectID, t.Dataset, t.Table}, ".")
}

type scorecardRow struct {
	Repo struct {
		Name string
	}
	Score float64
	Date  civil.Date
}

func (t *table) InsertScore(ctx context.Context, bq *bigquery.Client, repo string, score float64, date civil.Date) error {
	u := bq.DatasetInProject(t.ProjectID, t.Dataset).Table(t.Table).Inserter()

	row := &scorecardRow{
		Score: score,
		Date:  date,
	}
	row.Repo.Name = repo

	return u.Put(ctx, []*scorecardRow{row})
}

func (t *table) GetScores(ctx context.Context, bq *bigquery.Client, repos []string) ([]*scorecardResponse, error) {
	var resps []*scorecardResponse

	q := bq.Query(fmt.Sprintf(`
SELECT repo.name as repositoryname, score, date
FROM `+fmt.Sprintf("`%s`", t)+`
WHERE repo.name IN ('%s');
`,
		strings.Join(repos, "', '"),
	))

	it, err := q.Read(ctx)
	if err != nil {
		return resps, err
	}

	for {
		resp := &scorecardResponse{}
		err := it.Next(resp)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return resps, err
		}

		resps = append(resps, resp)
	}

	return resps, nil
}
