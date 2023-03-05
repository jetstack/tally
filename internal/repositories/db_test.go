package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/db"
	"github.com/jetstack/tally/internal/types"
)

type mockDB struct {
	getRepositories func(ctx context.Context, system, name string) ([]string, error)
}

func (d *mockDB) GetRepositories(ctx context.Context, system, name string) ([]string, error) {
	return d.getRepositories(ctx, system, name)
}

func TestDBMapperRepositories(t *testing.T) {
	type testCase struct {
		getRepos  func(ctx context.Context, system, name string) ([]string, error)
		wantRepos []string
		wantErr   error
	}
	testCases := map[string]func(t *testing.T) *testCase{
		"should return repos from GetRepositories": func(t *testing.T) *testCase {
			return &testCase{
				getRepos: func(ctx context.Context, system, name string) ([]string, error) {
					return []string{"a", "b", "c"}, nil
				},
				wantRepos: []string{"a", "b", "c"},
			}
		},
		"should return error from GetRepositories": func(t *testing.T) *testCase {
			wantErr := errors.New("test error")
			return &testCase{
				getRepos: func(ctx context.Context, system, name string) ([]string, error) {
					return []string{}, wantErr
				},
				wantErr: wantErr,
			}
		},
		"should ignore db.ErrNotFound": func(t *testing.T) *testCase {
			return &testCase{
				getRepos: func(ctx context.Context, system, name string) ([]string, error) {
					return []string{}, db.ErrNotFound
				},
				wantRepos: []string{},
			}
		},
	}
	for n, setup := range testCases {
		t.Run(n, func(t *testing.T) {
			tc := setup(t)

			gotRepos, err := DBMapper(&mockDB{tc.getRepos}).Repositories(context.Background(), types.Package{
				Type: "golang",
				Name: "foo/bar",
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
