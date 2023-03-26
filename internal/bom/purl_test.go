package bom

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/types"
)

func TestPackageRepositoriesFromPurl(t *testing.T) {
	testCases := []struct {
		purl                    string
		wantPackageRepositories *types.PackageRepositories
		wantErr                 error
	}{
		{
			purl: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9",
			wantPackageRepositories: &types.PackageRepositories{
				Package: types.Package{
					Type: "maven",
					Name: "org.hdrhistogram/HdrHistogram",
				},
			},
		},
		{
			purl: "pkg:golang/sigs.k8s.io/release-utils@v0.7.3",
			wantPackageRepositories: &types.PackageRepositories{
				Package: types.Package{
					Type: "golang",
					Name: "sigs.k8s.io/release-utils",
				},
			},
		},
		{
			purl: "pkg:golang/github.com/foo/bar@v0.7.3",
			wantPackageRepositories: &types.PackageRepositories{
				Package: types.Package{
					Type: "golang",
					Name: "github.com/foo/bar",
				},
				Repositories: []types.Repository{
					{
						Name: "github.com/foo/bar",
					},
				},
			},
		},
		{
			purl: "pkg:npm/zwitch@2.0.2",
			wantPackageRepositories: &types.PackageRepositories{
				Package: types.Package{
					Type: "npm",
					Name: "zwitch",
				},
			},
		},
		{
			purl: "pkg:cargo/getrandom@0.2.7",
			wantPackageRepositories: &types.PackageRepositories{
				Package: types.Package{
					Type: "cargo",
					Name: "getrandom",
				},
			},
		},
		{
			purl: "pkg:pypi/zope.interface@5.4.0",
			wantPackageRepositories: &types.PackageRepositories{
				Package: types.Package{
					Type: "pypi",
					Name: "zope.interface",
				},
			},
		},
		{
			purl: "pkg:pypi/foo.bar@5.4.0?vcs_url=git+git+ssh://git@github.com:foo/bar.git#v5.4.0",
			wantPackageRepositories: &types.PackageRepositories{
				Package: types.Package{
					Type: "pypi",
					Name: "foo.bar",
				},
				Repositories: []types.Repository{
					{
						Name: "github.com/foo/bar",
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		gotPkg, err := packageRepositoriesFromPurl(tc.purl)
		if !errors.Is(err, tc.wantErr) {
			t.Fatalf("unexpected error; wanted %s but got %s", tc.wantErr, err)
		}

		if diff := cmp.Diff(tc.wantPackageRepositories, gotPkg); diff != "" {
			t.Errorf("unexpected package:\n%s", diff)
		}

	}
}
