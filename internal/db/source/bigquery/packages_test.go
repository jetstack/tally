package bigquery

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/db"
	"google.golang.org/api/iterator"
)

type mockPackageWriter struct {
	addPackages func(context.Context, ...db.Package) error
}

func (w *mockPackageWriter) AddPackages(ctx context.Context, pkgs ...db.Package) error {
	return w.addPackages(ctx, pkgs...)
}

type mockPackageIterator struct {
	rows []pkgRow
	idx  int
	err  error
}

func (i *mockPackageIterator) Next(any interface{}) error {
	if i.err != nil {
		return i.err
	}
	if len(i.rows) < i.idx+1 {
		return iterator.Done
	}

	if row, ok := any.(*pkgRow); ok {
		if row == nil {
			row = &pkgRow{}
		}
		row.System = i.rows[i.idx].System
		row.Name = i.rows[i.idx].Name
		row.Repository = i.rows[i.idx].Repository
	}

	i.idx++

	return nil
}

func TestPackageSourceUpdate(t *testing.T) {
	testCases := map[string]struct {
		newSrc  func(t *testing.T) *pkgSrc
		wantErr bool
	}{
		// Test that Update passes all the returned rows to AddPackages
		"happy path": {
			newSrc: func(t *testing.T) *pkgSrc {
				return &pkgSrc{
					read: func(ctx context.Context, q string) (bqRowIterator, error) {
						return &mockPackageIterator{
							rows: []pkgRow{
								{
									System:     "GO",
									Name:       "foo",
									Repository: "github.com/foo/bar",
								},
								{
									System:     "NPM",
									Name:       "bar",
									Repository: "github.com/bar/foo",
								},
								{
									System:     "CARGO",
									Name:       "baz",
									Repository: "github.com/baz/foo",
								},
							},
						}, nil

					},
					db: &mockPackageWriter{
						addPackages: func(ctx context.Context, gotPackages ...db.Package) error {
							wantPackages := []db.Package{
								{
									System:     "GO",
									Name:       "foo",
									Repository: "github.com/foo/bar",
								},
								{
									System:     "NPM",
									Name:       "bar",
									Repository: "github.com/bar/foo",
								},
								{
									System:     "CARGO",
									Name:       "baz",
									Repository: "github.com/baz/foo",
								},
							}
							if diff := cmp.Diff(wantPackages, gotPackages); diff != "" {
								t.Fatalf("unexpected pkgs:\n%s", diff)
							}

							return nil
						},
					},
					batchSize: 5000000,
				}
			},
		},
		// Test that Update splits the packages up into batches, according
		// to the configured batchSize
		"batch": {
			newSrc: func(t *testing.T) *pkgSrc {
				i := 0
				return &pkgSrc{
					read: func(ctx context.Context, q string) (bqRowIterator, error) {
						return &mockPackageIterator{
							rows: []pkgRow{
								{
									System:     "GO",
									Name:       "foo",
									Repository: "github.com/foo/bar",
								},
								{
									System:     "NPM",
									Name:       "bar",
									Repository: "github.com/bar/foo",
								},
								{
									System:     "CARGO",
									Name:       "baz",
									Repository: "github.com/baz/foo",
								},
							},
						}, nil

					},
					db: &mockPackageWriter{
						addPackages: func(ctx context.Context, gotPackages ...db.Package) error {
							wantPackages := [][]db.Package{
								{
									{
										System:     "GO",
										Name:       "foo",
										Repository: "github.com/foo/bar",
									},
									{
										System:     "NPM",
										Name:       "bar",
										Repository: "github.com/bar/foo",
									},
								},
								{
									{
										System:     "CARGO",
										Name:       "baz",
										Repository: "github.com/baz/foo",
									},
								},
							}
							if diff := cmp.Diff(wantPackages[i], gotPackages); diff != "" {
								t.Fatalf("unexpected pkgs:\n%s", diff)
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
		// from BigQuery. AddPackages shouldn't be called.
		"read error": {
			newSrc: func(t *testing.T) *pkgSrc {
				return &pkgSrc{
					read: func(ctx context.Context, q string) (bqRowIterator, error) {
						return nil, fmt.Errorf("error")
					},
					db: &mockPackageWriter{
						addPackages: func(ctx context.Context, gotPackages ...db.Package) error {
							t.Fatalf("unexpected AddPackages call")

							return nil
						},
					},
					batchSize: 5000000,
				}
			},
			wantErr: true,
		},
		// Update should return an error when there's an error calling
		// AddPackages
		"write error": {
			newSrc: func(t *testing.T) *pkgSrc {
				return &pkgSrc{
					read: func(ctx context.Context, q string) (bqRowIterator, error) {
						return &mockPackageIterator{
							rows: []pkgRow{
								{
									System:     "CARGO",
									Name:       "baz",
									Repository: "github.com/baz/foo",
								},
							},
						}, nil

					},
					db: &mockPackageWriter{
						addPackages: func(ctx context.Context, gotPackages ...db.Package) error {
							return fmt.Errorf("error")
						},
					},
					batchSize: 5000000,
				}
			},
			wantErr: true,
		},
		// Update should return an error when there's an error calling
		// AddPackages
		"iterator error": {
			newSrc: func(t *testing.T) *pkgSrc {
				return &pkgSrc{
					read: func(ctx context.Context, q string) (bqRowIterator, error) {
						return &mockPackageIterator{err: fmt.Errorf("error")}, nil
					},
					db: &mockPackageWriter{
						addPackages: func(ctx context.Context, gotPackages ...db.Package) error {
							t.Fatalf("unexpected AddPackages call")

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
