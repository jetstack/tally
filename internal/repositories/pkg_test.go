package repositories

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/types"
)

func TestPackageMapperRepositories(t *testing.T) {
	testCases := map[string]struct {
		pkg       types.Package
		wantRepos []string
	}{
		"should extract repository Go package hosted on github.com": {
			pkg: types.Package{
				System: "GO",
				Name:   "github.com/foo/bar",
			},
			wantRepos: []string{"github.com/foo/bar"},
		},
		"shouldn't extract repository from Go package hosted on gitlab.com": {
			pkg: types.Package{
				System: "GO",
				Name:   "gitlab.com/foo/bar",
			},
			wantRepos: []string{},
		},
		"shouldn't extract repository from non-Go package": {
			pkg: types.Package{
				System: "NPM",
				Name:   "github.com/foo/bar",
			},
			wantRepos: []string{},
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			gotRepos, err := PackageMapper.Repositories(context.Background(), tc.pkg)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(tc.wantRepos, gotRepos); diff != "" {
				t.Errorf("unexpected repos:\n%s", diff)
			}
		})
	}
}
