package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/types"
)

type mockBOM struct {
	repositories func(pkg types.Package) ([]string, error)
}

func (m *mockBOM) Packages() ([]types.Package, error) {
	return nil, nil
}

func (m *mockBOM) Repositories(pkg types.Package) ([]string, error) {
	return m.repositories(pkg)
}

func TestBOMMapperRepositories(t *testing.T) {
	type testCase struct {
		repoFn    func(types.Package) ([]string, error)
		wantRepos []string
		wantErr   error
	}
	testCases := map[string]func(t *testing.T) *testCase{
		"should return repositories retrieved from BOM": func(t *testing.T) *testCase {
			wantRepos := []string{"a", "b"}
			return &testCase{
				repoFn: func(types.Package) ([]string, error) {
					return wantRepos, nil
				},
				wantRepos: wantRepos,
			}
		},
		"should return error from BOM": func(t *testing.T) *testCase {
			wantErr := errors.New("repositories error")
			return &testCase{
				repoFn: func(types.Package) ([]string, error) {
					return []string{}, wantErr
				},
				wantErr: wantErr,
			}
		},
	}
	for n, setup := range testCases {
		t.Run(n, func(t *testing.T) {
			tc := setup(t)
			bom := &mockBOM{tc.repoFn}
			gotRepos, err := BOMMapper(bom).Repositories(context.Background(), types.Package{
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
