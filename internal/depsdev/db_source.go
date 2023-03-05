package depsdev

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/jetstack/tally/internal/db"
	"google.golang.org/api/iterator"
)

// bqRead executes a bigquery query and returns an iterator
type bqRead func(context.Context, string) (bqRowIterator, error)

// bqRowIterator implements the methods of bigquery.RowIterator that we're using
type bqRowIterator interface {
	Next(interface{}) error
}

type src struct {
	read      bqRead
	batchSize int
}

type row struct {
	System     string `bigquery:"system"`
	Name       string `bigquery:"name"`
	Repository string `bigquery:"repository"`
}

// NewDBSource returns a new source that fetches the package -> repository
// relationships from the deps.dev dataset and inserts them into the database
func NewDBSource(bq *bigquery.Client) db.Source {
	return &src{
		read: func(ctx context.Context, qs string) (bqRowIterator, error) {
			return bq.Query(qs).Read(ctx)
		},
		batchSize: 500000,
	}
}

// String is the name of the source
func (s *src) String() string {
	return "deps.dev"
}

// Update finds the repository for the latest version of every Github-hosted
// package in the deps.dev dataset and adds it to the database.
func (s *src) Update(ctx context.Context, w db.Writer) error {
	qs := `
        SELECT DISTINCT t1.System, t1.Name, CONCAT('github.com/', t1.ProjectName) as repository
        FROM  ` + "`bigquery-public-data.deps_dev_v1.PackageVersionToProjectLatest`" + ` t1
        INNER JOIN (
          SELECT System, Name, Version 
          FROM ` + "`bigquery-public-data.deps_dev_v1.PackageVersionsLatest`" + ` t2
          WHERE VersionInfo.Ordinal = (
            SELECT MAX(VersionInfo.Ordinal) 
            FROM ` + "`bigquery-public-data.deps_dev_v1.PackageVersionsLatest`" + ` t3
            WHERE t2.System = t3.System AND t2.Name = t3.Name
        )) t4 ON t1.System = t4.System AND t1.Name = t4.Name AND t1.Version = t4.Version
        WHERE t1.ProjectType = 'GITHUB' AND REGEXP_CONTAINS(t1.ProjectName, '^.+/.+$')
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
		var pkgs []db.Package

		// Insert after each batch of rows to avoid storing the
		// entire dataset in memory
		for len(pkgs) < s.batchSize {
			var r row
			err := it.Next(&r)
			if err == iterator.Done {
				done = true
				break
			}
			if err != nil {
				return fmt.Errorf("calling Next on iterator: %w", err)
			}
			i++

			pkgs = append(pkgs, db.Package{
				Name:       r.Name,
				System:     r.System,
				Repository: r.Repository,
			})
		}

		if err := w.AddPackages(ctx, pkgs...); err != nil {
			return fmt.Errorf("adding packages: %w", err)
		}

		pkgs = nil
	}
	return nil
}
