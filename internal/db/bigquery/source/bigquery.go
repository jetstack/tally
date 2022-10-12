package bigquery

import "context"

// bqRead executes a bigquery query and returns an iterator
type bqRead func(context.Context, string) (bqRowIterator, error)

// bqRowIterator implements the methods of bigquery.RowIterator that we're using
type bqRowIterator interface {
	Next(interface{}) error
}
