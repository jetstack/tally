package bom

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPackageFromPurl(t *testing.T) {
	testCases := []struct {
		purl    string
		wantPkg *Package
		wantErr error
	}{
		{
			purl: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9",
			wantPkg: &Package{
				Type: "maven",
				Name: "org.hdrhistogram/HdrHistogram",
			},
		},
		{
			purl: "pkg:golang/sigs.k8s.io/release-utils@v0.7.3",
			wantPkg: &Package{
				Type: "golang",
				Name: "sigs.k8s.io/release-utils",
			},
		},
		{
			purl: "pkg:golang/github.com/foo/bar@v0.7.3",
			wantPkg: &Package{
				Type: "golang",
				Name: "github.com/foo/bar",
				Repositories: []string{
					"github.com/foo/bar",
				},
			},
		},
		{
			purl: "pkg:npm/zwitch@2.0.2",
			wantPkg: &Package{
				Type: "npm",
				Name: "zwitch",
			},
		},
		{
			purl: "pkg:cargo/getrandom@0.2.7",
			wantPkg: &Package{
				Type: "cargo",
				Name: "getrandom",
			},
		},
		{
			purl: "pkg:pypi/zope.interface@5.4.0",
			wantPkg: &Package{
				Type: "pypi",
				Name: "zope.interface",
			},
		},
	}
	for _, tc := range testCases {
		gotPkg, err := packageFromPurl(tc.purl)
		if !errors.Is(err, tc.wantErr) {
			t.Fatalf("unexpected error; wanted %s but got %s", tc.wantErr, err)
		}

		if diff := cmp.Diff(tc.wantPkg, gotPkg); diff != "" {
			t.Errorf("unexpected package:\n%s", diff)
		}

	}
}
