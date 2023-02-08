package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/types"
)

type mockMapper struct {
	repositories func(ctx context.Context, pkg types.Package) ([]string, error)
}

func (m *mockMapper) Repositories(ctx context.Context, pkg types.Package) ([]string, error) {
	return m.repositories(ctx, pkg)
}

func TestMultiMapperRepositories(t *testing.T) {
	type testCase struct {
		pkg       types.Package
		repoFns   []func(ctx context.Context, pkg types.Package) ([]string, error)
		wantRepos []string
		wantErr   error
	}
	testCases := map[string]func(t *testing.T) *testCase{
		"should iterate through all mappers, regardless of whether they return anything": func(t *testing.T) *testCase {
			return &testCase{
				repoFns: []func(ctx context.Context, pkg types.Package) ([]string, error){
					func(ctx context.Context, pkg types.Package) ([]string, error) {
						return []string{}, nil
					},
					func(ctx context.Context, pkg types.Package) ([]string, error) {
						return []string{}, nil
					},
					func(ctx context.Context, pkg types.Package) ([]string, error) {
						return []string{"a"}, nil
					},
				},
				wantRepos: []string{"a"},
			}
		},
		"should aggregate repositories from multiple mappers": func(t *testing.T) *testCase {
			return &testCase{
				repoFns: []func(ctx context.Context, pkg types.Package) ([]string, error){
					func(ctx context.Context, pkg types.Package) ([]string, error) {
						return []string{"a", "b"}, nil
					},
					func(ctx context.Context, pkg types.Package) ([]string, error) {
						return []string{}, nil
					},
					func(ctx context.Context, pkg types.Package) ([]string, error) {
						return []string{"c"}, nil
					},
				},
				wantRepos: []string{"a", "b", "c"},
			}
		},
		"should deduplicate repositories from different mappers": func(t *testing.T) *testCase {
			return &testCase{
				repoFns: []func(ctx context.Context, pkg types.Package) ([]string, error){
					func(ctx context.Context, pkg types.Package) ([]string, error) {
						return []string{"a", "b"}, nil
					},
					func(ctx context.Context, pkg types.Package) ([]string, error) {
						return []string{"c", "b"}, nil
					},
					func(ctx context.Context, pkg types.Package) ([]string, error) {
						return []string{"a"}, nil
					},
				},
				wantRepos: []string{"a", "b", "c"},
			}
		},
		"should deduplicate repositories from the same mapper": func(t *testing.T) *testCase {
			return &testCase{
				repoFns: []func(ctx context.Context, pkg types.Package) ([]string, error){
					func(ctx context.Context, pkg types.Package) ([]string, error) {
						return []string{"a", "b", "c", "c", "a"}, nil
					},
				},
				wantRepos: []string{"a", "b", "c"},
			}
		},
		"should pass the same package to each mapper": func(t *testing.T) *testCase {
			var wantPkg *types.Package
			return &testCase{
				repoFns: []func(ctx context.Context, pkg types.Package) ([]string, error){
					func(ctx context.Context, pkg types.Package) ([]string, error) {
						wantPkg = &pkg
						return []string{}, nil
					},
					func(ctx context.Context, gotPkg types.Package) ([]string, error) {
						if !wantPkg.Equals(gotPkg) {
							t.Fatalf("unexpected package in second mapper; want %v got %v", wantPkg, gotPkg)
						}
						return []string{}, nil
					},
					func(ctx context.Context, gotPkg types.Package) ([]string, error) {
						if !wantPkg.Equals(gotPkg) {
							t.Fatalf("unexpected package in third mapper; want %v got %v", wantPkg, gotPkg)
						}
						return []string{"a"}, nil
					},
				},
				wantRepos: []string{"a"},
			}
		},
		"should return an error from any of the mappers": func(t *testing.T) *testCase {
			wantErr := errors.New("mapper error")
			return &testCase{
				repoFns: []func(ctx context.Context, pkg types.Package) ([]string, error){
					func(ctx context.Context, pkg types.Package) ([]string, error) {
						return []string{"a", "b"}, nil
					},
					func(ctx context.Context, pkg types.Package) ([]string, error) {
						return []string{}, wantErr
					},
					func(ctx context.Context, pkg types.Package) ([]string, error) {
						t.Errorf("unexpected call to third mapper")
						return []string{"a"}, nil
					},
				},
				wantRepos: []string{"a", "b", "c"},
				wantErr:   wantErr,
			}
		},
	}
	for n, setup := range testCases {
		t.Run(n, func(t *testing.T) {
			tc := setup(t)

			var mappers []Mapper
			for _, fn := range tc.repoFns {
				mappers = append(mappers, &mockMapper{fn})
			}

			gotRepos, err := From(mappers...).Repositories(context.Background(), types.Package{
				System: "GO",
				Name:   "foo/bar",
			})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("unexpected error: %s", err)
			}

			if tc.wantErr != nil {
				return
			}

			if diff := cmp.Diff(tc.wantRepos, gotRepos); diff != "" {
				t.Errorf("unexpected repositories:\n%s", diff)
			}
		})
	}
}
