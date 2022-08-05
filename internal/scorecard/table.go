package scorecard

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"google.golang.org/api/iterator"
)

// Row is a row in the scorecard table
type Row struct {
	Repo  Repo       `bigquery:"repo"`
	Score float64    `bigquery:"score"`
	Date  civil.Date `bigquery:"date"`
}

// Repo is a RECORD that holds repository information
type Repo struct {
	Name string `bigquery:"name"`
}

// Table is a table holding scorecard information
type Table interface {
	Create(context.Context) error
	SelectWhereRepositoryIn(context.Context, []string) ([]*Row, error)
	Insert(context.Context, *Row) error
	String() string
}

type table struct {
	bq        *bigquery.Client
	projectID string
	dataset   string
	tableName string
}

// NewTable returns a table from the given reference
func NewTable(bq *bigquery.Client, tableRef string) (Table, error) {
	parts := strings.Split(tableRef, ".")
	if len(parts) == 2 {
		projectID := bq.Project()
		if projectID == "" {
			return nil, fmt.Errorf("project id is not provided in reference and it can't be retrieved from the client")
		}
		return &table{
			projectID: projectID,
			dataset:   parts[0],
			tableName: parts[1],
			bq:        bq,
		}, nil
	}
	if len(parts) == 3 {
		return &table{
			projectID: parts[0],
			dataset:   parts[1],
			tableName: parts[2],
			bq:        bq,
		}, nil
	}

	return nil, fmt.Errorf("invalid table reference: %s", tableRef)
}

// Create the table
func (t *table) Create(ctx context.Context) error {
	schema, err := bigquery.InferSchema(Row{})
	if err != nil {
		return fmt.Errorf("inferring schema: %w", err)
	}

	err = t.bq.DatasetInProject(t.projectID, t.dataset).
		Table(t.tableName).
		Create(ctx, &bigquery.TableMetadata{Schema: schema})
	if err != nil {
		return fmt.Errorf("creating table: %w", err)
	}

	return nil
}

// String returns a string representation of the table in the form:
// <project-id>.<dataset-name>.<table-name>
func (t *table) String() string {
	return strings.Join([]string{t.projectID, t.dataset, t.tableName}, ".")
}

// Insert a row into the table
func (t *table) Insert(ctx context.Context, row *Row) error {
	u := t.bq.DatasetInProject(t.projectID, t.dataset).Table(t.tableName).Inserter()

	return u.Put(ctx, []*Row{row})
}

// SelectWhereRepositoryIn selects rows for the provided list of repositories
func (t *table) SelectWhereRepositoryIn(ctx context.Context, repos []string) ([]*Row, error) {
	var rows []*Row

	q := t.bq.Query(fmt.Sprintf(`
SELECT repo, score, date
FROM `+fmt.Sprintf("`%s`", t)+`
WHERE repo.name IN ('%s');
`,
		strings.Join(repos, "', '"),
	))

	it, err := q.Read(ctx)
	if err != nil {
		return rows, err
	}

	for {
		row := &Row{}
		err := it.Next(row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return rows, err
		}

		rows = append(rows, row)
	}

	return rows, nil
}
