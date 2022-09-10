package bigquery

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/db"
	"google.golang.org/api/iterator"
)

type mockScoreWriter struct {
	addScores func(context.Context, ...db.Score) error
}

func (w *mockScoreWriter) AddScores(ctx context.Context, scores ...db.Score) error {
	return w.addScores(ctx, scores...)
}

type mockScoreIterator struct {
	rows []scoreRow
	idx  int
	err  error
}

func (i *mockScoreIterator) Next(any interface{}) error {
	if i.err != nil {
		return i.err
	}
	if len(i.rows) < i.idx+1 {
		return iterator.Done
	}

	if row, ok := any.(*scoreRow); ok {
		if row == nil {
			row = &scoreRow{}
		}
		row.Repository = i.rows[i.idx].Repository
		row.Score = i.rows[i.idx].Score
	}

	i.idx++

	return nil
}

func TestScoreSourceUpdate(t *testing.T) {
	testCases := map[string]struct {
		newSrc  func(t *testing.T) *scoreSrc
		wantErr bool
	}{
		// Test that Update passes all the returned rows to AddScores
		"happy path": {
			newSrc: func(t *testing.T) *scoreSrc {
				return &scoreSrc{
					read: func(ctx context.Context, q string) (bqRowIterator, error) {
						return &mockScoreIterator{
							rows: []scoreRow{
								{
									Repository: "github.com/foo/bar",
									Score:      7.1,
								},
								{
									Repository: "github.com/bar/foo",
									Score:      3.2,
								},
								{
									Repository: "github.com/baz/foo",
									Score:      8.4,
								},
							},
						}, nil

					},
					db: &mockScoreWriter{
						addScores: func(ctx context.Context, gotScores ...db.Score) error {
							wantScores := []db.Score{
								{
									Repository: "github.com/foo/bar",
									Score:      7.1,
								},
								{
									Repository: "github.com/bar/foo",
									Score:      3.2,
								},
								{
									Repository: "github.com/baz/foo",
									Score:      8.4,
								},
							}
							if diff := cmp.Diff(wantScores, gotScores); diff != "" {
								t.Fatalf("unexpected scores:\n%s", diff)
							}

							return nil
						},
					},
					batchSize: 5000000,
				}
			},
		},
		// Test that Update splits the scores up into batches, according
		// to the configured batchSize
		"batch": {
			newSrc: func(t *testing.T) *scoreSrc {
				i := 0
				return &scoreSrc{
					read: func(ctx context.Context, q string) (bqRowIterator, error) {
						return &mockScoreIterator{
							rows: []scoreRow{
								{
									Repository: "github.com/foo/bar",
									Score:      7.1,
								},
								{
									Repository: "github.com/bar/foo",
									Score:      3.2,
								},
								{
									Repository: "github.com/baz/foo",
									Score:      8.4,
								},
							},
						}, nil

					},
					db: &mockScoreWriter{
						addScores: func(ctx context.Context, gotScores ...db.Score) error {
							wantScores := [][]db.Score{
								{
									{
										Repository: "github.com/foo/bar",
										Score:      7.1,
									},
									{
										Repository: "github.com/bar/foo",
										Score:      3.2,
									},
								},
								{
									{
										Repository: "github.com/baz/foo",
										Score:      8.4,
									},
								},
							}
							if diff := cmp.Diff(wantScores[i], gotScores); diff != "" {
								t.Fatalf("unexpected scores:\n%s", diff)
							}

							i++

							return nil
						},
					},
					batchSize: 2,
				}
			},
		},
		// Update should return an error when there's an error reading
		// from BigQuery. AddScores shouldn't be called.
		"read error": {
			newSrc: func(t *testing.T) *scoreSrc {
				return &scoreSrc{
					read: func(ctx context.Context, q string) (bqRowIterator, error) {
						return nil, fmt.Errorf("error")
					},
					db: &mockScoreWriter{
						addScores: func(ctx context.Context, gotScores ...db.Score) error {
							t.Fatalf("unexpected AddScores call")

							return nil
						},
					},
					batchSize: 5000000,
				}
			},
			wantErr: true,
		},
		// Update should return an error when there's an error calling
		// AddScores
		"write error": {
			newSrc: func(t *testing.T) *scoreSrc {
				return &scoreSrc{
					read: func(ctx context.Context, q string) (bqRowIterator, error) {
						return &mockScoreIterator{
							rows: []scoreRow{
								{
									Repository: "github.com/foo/bar",
									Score:      7.1,
								},
							},
						}, nil

					},
					db: &mockScoreWriter{
						addScores: func(ctx context.Context, gotScores ...db.Score) error {
							return fmt.Errorf("error")
						},
					},
					batchSize: 5000000,
				}
			},
			wantErr: true,
		},
		// Update should return an error when there's an error calling
		// Next on the iterator
		"iterator error": {
			newSrc: func(t *testing.T) *scoreSrc {
				return &scoreSrc{
					read: func(ctx context.Context, q string) (bqRowIterator, error) {
						return &mockScoreIterator{err: fmt.Errorf("error")}, nil
					},
					db: &mockScoreWriter{
						addScores: func(ctx context.Context, gotScores ...db.Score) error {
							t.Fatalf("unexpected AddScores call")

							return nil
						},
					},
					batchSize: 5000000,
				}
			},
			wantErr: true,
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			src := tc.newSrc(t)

			err := src.Update(context.Background())
			if err != nil && !tc.wantErr {
				t.Fatalf("unexpected error calling Update: %s", err)
			}
			if err == nil && tc.wantErr {
				t.Fatalf("expected error but got nil")
			}
		})
	}
}
