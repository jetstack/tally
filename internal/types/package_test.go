package types

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPackages_Add(t *testing.T) {
	testCases := map[string]struct {
		pkgs     Packages
		wantPkgs Packages
		pkg      *Package
	}{
		"new package": {
			pkgs: Packages{
				{
					System: "GO",
					Name:   "foobar",
				},
			},
			pkg: &Package{
				System: "MAVEN",
				Name:   "foo:bar",
			},
			wantPkgs: Packages{
				{
					System: "GO",
					Name:   "foobar",
				},
				{
					System: "MAVEN",
					Name:   "foo:bar",
				},
			},
		},
		"new package with repositories": {
			pkgs: Packages{
				{
					System: "GO",
					Name:   "foobar",
				},
			},
			pkg: &Package{
				System: "MAVEN",
				Name:   "foo:bar",
				Repositories: []string{
					"github.com/foo/bar",
				},
			},
			wantPkgs: Packages{
				{
					System: "GO",
					Name:   "foobar",
				},
				{
					System: "MAVEN",
					Name:   "foo:bar",
					Repositories: []string{
						"github.com/foo/bar",
					},
				},
			},
		},
		"ignore duplicate": {
			pkgs: Packages{
				{
					System: "GO",
					Name:   "foobar",
				},
			},
			pkg: &Package{
				System: "GO",
				Name:   "foobar",
			},
			wantPkgs: Packages{
				{
					System: "GO",
					Name:   "foobar",
				},
			},
		},
		"ignore duplicate repo": {
			pkgs: Packages{
				{
					System: "GO",
					Name:   "foobar",
					Repositories: []string{
						"github.com/foo/bar",
					},
				},
			},
			pkg: &Package{
				System: "GO",
				Name:   "foobar",
				Repositories: []string{
					"github.com/foo/bar",
					"github.com/bar/foo",
				},
			},
			wantPkgs: Packages{
				{
					System: "GO",
					Name:   "foobar",
					Repositories: []string{
						"github.com/foo/bar",
						"github.com/bar/foo",
					},
				},
			},
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			tc.pkgs.Add(tc.pkg)

			if diff := cmp.Diff(tc.pkgs, tc.wantPkgs); diff != "" {
				t.Fatalf("unexpected packages:\n%s", diff)
			}
		})
	}
}

func TestPackage_AddRepositories(t *testing.T) {
	testCases := map[string]struct {
		pkg     *Package
		wantPkg *Package
		repos   []string
	}{
		"new repo": {
			pkg: &Package{
				System: "GO",
				Name:   "foobar",
			},
			repos: []string{
				"github.com/foo/bar",
			},
			wantPkg: &Package{
				System: "GO",
				Name:   "foobar",
				Repositories: []string{
					"github.com/foo/bar",
				},
			},
		},
		"append to existing repos": {
			pkg: &Package{
				System: "GO",
				Name:   "foobar",
				Repositories: []string{
					"github.com/foo/bar",
				},
			},
			repos: []string{
				"github.com/bar/foo",
			},
			wantPkg: &Package{
				System: "GO",
				Name:   "foobar",
				Repositories: []string{
					"github.com/foo/bar",
					"github.com/bar/foo",
				},
			},
		},
		"ignore duplicate repos": {
			pkg: &Package{
				System: "GO",
				Name:   "foobar",
				Repositories: []string{
					"github.com/foo/bar",
				},
			},
			repos: []string{
				"github.com/foo/bar",
				"github.com/bar/foo",
			},
			wantPkg: &Package{
				System: "GO",
				Name:   "foobar",
				Repositories: []string{
					"github.com/foo/bar",
					"github.com/bar/foo",
				},
			},
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			tc.pkg.AddRepositories(tc.repos...)

			if diff := cmp.Diff(tc.pkg, tc.wantPkg); diff != "" {
				t.Fatalf("unexpected package:\n%s", diff)
			}
		})
	}

}

func TestPackage_Equals(t *testing.T) {
	testCases := map[string]struct {
		pkg1       *Package
		pkg2       *Package
		wantEquals bool
	}{
		"equal": {
			pkg1: &Package{
				System: "GO",
				Name:   "foobar",
			},
			pkg2: &Package{
				System: "GO",
				Name:   "foobar",
			},
			wantEquals: true,
		},
		"not equal": {
			pkg1: &Package{
				System: "GO",
				Name:   "foobar",
			},
			pkg2: &Package{
				System: "GO",
				Name:   "barfoo",
			},
		},
		"ignores missing repositories": {
			pkg1: &Package{
				System: "GO",
				Name:   "foobar",
				Repositories: []string{
					"github.com/foo/bar",
				},
			},
			pkg2: &Package{
				System: "GO",
				Name:   "foobar",
			},
			wantEquals: true,
		},
		"ignores different repositories": {
			pkg1: &Package{
				System: "GO",
				Name:   "foobar",
				Repositories: []string{
					"github.com/foo/bar",
				},
			},
			pkg2: &Package{
				System: "GO",
				Name:   "foobar",
				Repositories: []string{
					"github.com/bar/foo",
				},
			},
			wantEquals: true,
		},
		"nil package is not equal": {
			pkg1: &Package{
				System: "GO",
				Name:   "foobar",
				Repositories: []string{
					"github.com/foo/bar",
				},
			},
		},
		"both nil is equal": {
			wantEquals: true,
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			if equals := tc.pkg1.Equals(tc.pkg2); equals != tc.wantEquals {
				t.Fatalf("unexpected error comparing %v to %v; wanted %v but got %v", tc.pkg1, tc.pkg2, tc.wantEquals, equals)
			}
		})
	}
}
