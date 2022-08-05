package tally

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"
	"github.com/ribbybibby/tally/internal/scorecard"
)

// Dataset is a BigQuery dataset for storing generated scores
type Dataset interface {
	Create(context.Context) error
	ScorecardTable() (scorecard.Table, error)
	String() string
}

type dataset struct {
	bq        *bigquery.Client
	projectID string
	name      string
}

// NewDataset returns a new dataset from the given reference
func NewDataset(bq *bigquery.Client, ref string) (Dataset, error) {
	parts := strings.Split(ref, ".")
	if len(parts) == 1 {
		projectID := bq.Project()
		if projectID == "" {
			return nil, fmt.Errorf("project id is not provided in reference and it can't be retrieved from the client")
		}
		return &dataset{
			projectID: projectID,
			name:      parts[0],
			bq:        bq,
		}, nil
	}
	if len(parts) == 2 {
		return &dataset{
			projectID: parts[0],
			name:      parts[1],
			bq:        bq,
		}, nil
	}

	return nil, fmt.Errorf("invalid dataset reference: %s", ref)
}

// Create creates the dataset
func (d *dataset) Create(ctx context.Context) error {
	if err := d.bq.DatasetInProject(d.projectID, d.name).Create(ctx, nil); err != nil {
		return fmt.Errorf("creating dataset: %w", err)
	}

	scorecardTable, err := d.ScorecardTable()
	if err != nil {
		return fmt.Errorf("new scorecard table: %w", err)
	}

	if err := scorecardTable.Create(ctx); err != nil {
		return fmt.Errorf("creating scorecard table: %w", err)
	}

	return nil
}

// String returns a reference to the dataset in the form
// <project-id>.<dataset-name>
func (d *dataset) String() string {
	return strings.Join([]string{d.projectID, d.name}, ".")
}

// ScorecardTable returns the scorecard table in the dataset
func (d *dataset) ScorecardTable() (scorecard.Table, error) {
	return scorecard.NewTable(d.bq, strings.Join([]string{d.String(), "scorecard"}, "."))
}
