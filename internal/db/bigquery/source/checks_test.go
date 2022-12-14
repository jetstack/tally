package bigquery

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/db"
	"google.golang.org/api/iterator"
)

type mockDBWriter struct {
	addChecks   func(context.Context, ...db.Check) error
	addPackages func(context.Context, ...db.Package) error
	addScores   func(context.Context, ...db.Score) error
}

func (w *mockDBWriter) AddChecks(ctx context.Context, checks ...db.Check) error {
	if w.addChecks == nil {
		return nil
	}
	return w.addChecks(ctx, checks...)
}

func (w *mockDBWriter) AddPackages(ctx context.Context, pkgs ...db.Package) error {
	if w.addPackages == nil {
		return nil
	}
	return w.addPackages(ctx, pkgs...)
}

func (w *mockDBWriter) AddScores(ctx context.Context, scores ...db.Score) error {
	if w.addScores == nil {
		return nil
	}
	return w.addScores(ctx, scores...)
}

type mockCheckIterator struct {
	rows []checkRow
	idx  int
	err  error
}

func (i *mockCheckIterator) Next(any interface{}) error {
	if i.err != nil {
		return i.err
	}
	if len(i.rows) < i.idx+1 {
		return iterator.Done
	}

	if row, ok := any.(*checkRow); ok {
		if row == nil {
			row = &checkRow{}
		}
		row.Name = i.rows[i.idx].Name
		row.Repository = i.rows[i.idx].Repository
		row.Score = i.rows[i.idx].Score
	}

	i.idx++

	return nil
}

func TestCheckSourceUpdate(t *testing.T) {
	testCases := map[string]struct {
		newSrc      func(t *testing.T) *checkSrc
		newDBWriter func(t *testing.T) db.DBWriter
		wantErr     bool
	}{
		// Test that Update passes all the returned rows to AddChecks
		"happy path": {
			newDBWriter: func(t *testing.T) db.DBWriter {
				return &mockDBWriter{
					addChecks: func(ctx context.Context, gotChecks ...db.Check) error {
						wantChecks := []db.Check{
							{
								Name:       "foo",
								Repository: "github.com/foo/bar",
								Score:      7,
							},
							{
								Name:       "bar",
								Repository: "github.com/bar/foo",
								Score:      3,
							},
							{
								Name:       "foo",
								Repository: "github.com/bar/foo",
								Score:      8,
							},
						}
						if diff := cmp.Diff(wantChecks, gotChecks); diff != "" {
							t.Fatalf("unexpected checks:\n%s", diff)
						}

						return nil
					},
				}
			},
			newSrc: func(t *testing.T) *checkSrc {
				return &checkSrc{
					read: func(ctx context.Context, q string) (bqRowIterator, error) {
						return &mockCheckIterator{
							rows: []checkRow{
								{
									Name:       "foo",
									Repository: "github.com/foo/bar",
									Score:      7,
								},
								{
									Name:       "bar",
									Repository: "github.com/bar/foo",
									Score:      3,
								},
								{
									Name:       "foo",
									Repository: "github.com/bar/foo",
									Score:      8,
								},
							},
						}, nil

					},
					batchSize: 5000000,
				}
			},
		},
		// Test that Update splits the checks up into batches, according
		// to the configured batchSize
		"batch": {
			newDBWriter: func(t *testing.T) db.DBWriter {
				i := 0
				return &mockDBWriter{
					addChecks: func(ctx context.Context, gotChecks ...db.Check) error {
						wantChecks := [][]db.Check{
							{
								{
									Name:       "foo",
									Repository: "github.com/foo/bar",
									Score:      7,
								},
								{
									Name:       "bar",
									Repository: "github.com/bar/foo",
									Score:      3,
								},
							},
							{
								{
									Name:       "foo",
									Repository: "github.com/bar/foo",
									Score:      8,
								},
							},
						}
						if diff := cmp.Diff(wantChecks[i], gotChecks); diff != "" {
							t.Fatalf("unexpected checks:\n%s", diff)
						}

						i++

						return nil
					},
				}
			},
			newSrc: func(t *testing.T) *checkSrc {
				return &checkSrc{
					read: func(ctx context.Context, q string) (bqRowIterator, error) {
						return &mockCheckIterator{
							rows: []checkRow{
								{
									Name:       "foo",
									Repository: "github.com/foo/bar",
									Score:      7,
								},
								{
									Name:       "bar",
									Repository: "github.com/bar/foo",
									Score:      3,
								},
								{
									Name:       "foo",
									Repository: "github.com/bar/foo",
									Score:      8,
								},
							},
						}, nil

					},
					batchSize: 2,
				}
			},
		},
		// Update should return an error when there's an error reading
		// from BigQuery. AddChecks shouldn't be called.
		"read error": {
			newDBWriter: func(t *testing.T) db.DBWriter {
				return &mockDBWriter{
					addChecks: func(ctx context.Context, gotChecks ...db.Check) error {
						t.Fatalf("unexpected AddChecks call")

						return nil
					},
				}
			},
			newSrc: func(t *testing.T) *checkSrc {
				return &checkSrc{
					read: func(ctx context.Context, q string) (bqRowIterator, error) {
						return nil, fmt.Errorf("error")
					},
					batchSize: 5000000,
				}
			},
			wantErr: true,
		},
		// Update should return an error when there's an error calling
		// AddChecks
		"write error": {
			newDBWriter: func(t *testing.T) db.DBWriter {
				return &mockDBWriter{
					addChecks: func(ctx context.Context, gotChecks ...db.Check) error {
						return fmt.Errorf("error")
					},
				}

			},
			newSrc: func(t *testing.T) *checkSrc {
				return &checkSrc{
					read: func(ctx context.Context, q string) (bqRowIterator, error) {
						return &mockCheckIterator{
							rows: []checkRow{
								{
									Name:       "foo",
									Repository: "github.com/foo/bar",
									Score:      7,
								},
							},
						}, nil

					},
					batchSize: 5000000,
				}
			},
			wantErr: true,
		},
		// Update should return an error when there's an error calling
		// AddChecks
		"iterator error": {
			newDBWriter: func(t *testing.T) db.DBWriter {
				return &mockDBWriter{
					addChecks: func(ctx context.Context, gotChecks ...db.Check) error {
						t.Fatalf("unexpected AddChecks call")

						return nil
					},
				}
			},
			newSrc: func(t *testing.T) *checkSrc {
				return &checkSrc{
					read: func(ctx context.Context, q string) (bqRowIterator, error) {
						return &mockCheckIterator{err: fmt.Errorf("error")}, nil
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
			w := tc.newDBWriter(t)

			err := src.Update(context.Background(), w)
			if err != nil && !tc.wantErr {
				t.Fatalf("unexpected error calling Update: %s", err)
			}
			if err == nil && tc.wantErr {
				t.Fatalf("expected error but got nil")
			}
		})
	}
}
