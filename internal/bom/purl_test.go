package bom

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/types"
	"github.com/package-url/packageurl-go"
)

func TestPackageFromPurl(t *testing.T) {
	testCases := []struct {
		purl    string
		wantPkg *types.Package
		wantErr error
	}{
		{
			purl: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9",
			wantPkg: &types.Package{
				System: "MAVEN",
				Name:   "org.hdrhistogram:HdrHistogram",
			},
		},
		{
			purl: "pkg:golang/sigs.k8s.io/release-utils@v0.7.3",
			wantPkg: &types.Package{
				System: "GO",
				Name:   "sigs.k8s.io/release-utils",
			},
		},
		{
			purl: "pkg:npm/zwitch@2.0.2",
			wantPkg: &types.Package{
				System: "NPM",
				Name:   "zwitch",
			},
		},
		{
			purl: "pkg:cargo/getrandom@0.2.7",
			wantPkg: &types.Package{
				System: "CARGO",
				Name:   "getrandom",
			},
		},
		{
			purl: "pkg:pypi/zope.interface@5.4.0",
			wantPkg: &types.Package{
				System: "PYPI",
				Name:   "zope.interface",
			},
		},
		{
			purl:    "pkg:nuget/EnterpriseLibrary.Common@6.0.1304",
			wantErr: ErrUnsupportedPackageType,
		},
	}
	for _, tc := range testCases {
		purl, err := packageurl.FromString(tc.purl)
		if err != nil {
			t.Fatalf("unexpected error parsing purl: %s", err)
		}

		gotPkg, err := packageFromPurl(purl)
		if !errors.Is(err, tc.wantErr) {
			t.Fatalf("unexpected error; wanted %s but got %s", tc.wantErr, err)
		}

		if diff := cmp.Diff(tc.wantPkg, gotPkg); diff != "" {
			t.Errorf("unexpected package:\n%s", diff)
		}

	}
}
