package bigquery

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/googleapi"
)

// DB is a database implementation that stores scores in a bigquery dataset
type DB struct {
	bq         *bigquery.Client
	dataset    *bigquery.Dataset
	scoreTable *bigquery.Table
}

// NewDB returns a database that stores scores in a bigquery dataset
func NewDB(ctx context.Context, bq *bigquery.Client, ref string) (*DB, error) {
	parts := strings.Split(ref, ".")
	if len(parts) == 1 {
		projectID := bq.Project()
		if projectID == "" {
			return nil, fmt.Errorf("project id is not provided in reference and it can't be retrieved from the client")
		}
		dataset := bq.DatasetInProject(projectID, parts[0])
		scoreTable := dataset.Table("scorecard")
		return &DB{
			dataset:    dataset,
			scoreTable: scoreTable,
			bq:         bq,
		}, nil
	}
	if len(parts) == 3 {
		dataset := bq.DatasetInProject(parts[0], parts[1])
		scoreTable := dataset.Table("scorecard")
		return &DB{
			dataset:    dataset,
			scoreTable: scoreTable,
			bq:         bq,
		}, nil
	}

	return nil, fmt.Errorf("invalid table reference: %s", ref)
}

// Initialize creates the dataset
func (d *DB) Initialize(ctx context.Context) error {
	// Create the dataset
	if err := createDataset(ctx, d.dataset); err != nil {
		return fmt.Errorf("creating dataset: %w", err)
	}

	// Create the scorecard table
	schema, err := bigquery.InferSchema(scoreRow{})
	if err != nil {
		return fmt.Errorf("inferring schema: %s", err)
	}
	if err := createTable(ctx, d.scoreTable, &bigquery.TableMetadata{Schema: schema}); err != nil {
		return fmt.Errorf("creating scorecard table: %w", err)
	}

	return nil
}

func createDataset(ctx context.Context, dataset *bigquery.Dataset) error {
	if err := dataset.Create(ctx, nil); err != nil {
		if gErr, ok := err.(*googleapi.Error); ok {
			// Ignore already exists error
			if gErr.Code == 409 {
				return nil
			}
		}
		return fmt.Errorf("creating dataset: %w", err)
	}

	return nil
}

func createTable(ctx context.Context, table *bigquery.Table, metadata *bigquery.TableMetadata) error {
	if err := table.Create(ctx, metadata); err != nil {
		if gErr, ok := err.(*googleapi.Error); ok {
			// Ignore already exists error
			if gErr.Code == 409 {
				return nil
			}
		}
		return fmt.Errorf("creating tablet: %w", err)
	}

	return nil
}
