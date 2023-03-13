package bom

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseSyftBOM(t *testing.T) {
	testCases := map[string]struct {
		path    string
		wantBOM *syftJSON
		wantErr bool
	}{
		"json is parsed successfully": {
			path: "testdata/syft.json",
			wantBOM: &syftJSON{
				Artifacts: []syftArtifact{
					{
						Purl: "pkg:golang/foo/bar@v0.2.5",
					},
					{
						Purl: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9",
					},
				},
			},
		},
		"error is returned when parsing invalid json": {
			path:    "testdata/syft.json.invalid",
			wantErr: true,
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			r, err := os.Open(tc.path)
			if err != nil {
				t.Fatalf("unexpected error opening file: %s", err)
			}
			defer r.Close()

			gotBOM, err := ParseSyftBOM(r)
			if err != nil && !tc.wantErr {
				t.Fatalf("unexpected error parsing BOM: %s", err)
			}
			if err == nil && tc.wantErr {
				t.Fatalf("expected parsing BOM but got nil")
			}

			if tc.wantErr {
				return
			}

			if diff := cmp.Diff(tc.wantBOM, gotBOM); diff != "" {
				t.Errorf("unexpected BOM:\n%s", diff)
			}
		})
	}
}

func TestPackageFromSyftBOM(t *testing.T) {
	testCases := map[string]struct {
		bom          *syftJSON
		wantPackages []Package
	}{
		"an error should not be produced for an empty BOM": {
			bom: &syftJSON{},
		},
		"components without a Purl should be ignored": {
			bom: &syftJSON{
				Artifacts: []syftArtifact{
					{},
				},
			},
		},
		"duplicate packages should be ignored": {
			bom: &syftJSON{
				Artifacts: []syftArtifact{
					{
						Purl: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.8",
					},
					{
						Purl: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9",
					},
					{
						Purl: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9",
					},
				},
			},
			wantPackages: []Package{
				{
					Type: "maven",
					Name: "org.hdrhistogram/HdrHistogram",
				},
			},
		},
		"all supported types should be discovered": {
			bom: &syftJSON{
				Artifacts: []syftArtifact{
					{
						Purl: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9",
					},
					{
						Purl: "pkg:golang/sigs.k8s.io/release-utils@v0.7.3",
					},
					{
						Purl: "pkg:npm/zwitch@2.0.2",
					},
					{
						Purl: "pkg:cargo/getrandom@0.2.7",
					},
					{
						Purl: "pkg:pypi/zope.interface@5.4.0",
					},
				},
			},
			wantPackages: []Package{
				{
					Type: "maven",
					Name: "org.hdrhistogram/HdrHistogram",
				},
				{
					Type: "golang",
					Name: "sigs.k8s.io/release-utils",
				},
				{
					Type: "npm",
					Name: "zwitch",
				},
				{
					Type: "cargo",
					Name: "getrandom",
				},
				{
					Type: "pypi",
					Name: "zope.interface",
				},
			},
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {

			gotPackages, err := PackagesFromSyftBOM(tc.bom)
			if err != nil {
				t.Fatalf("unexpected error getting packages from bom: %s", err)
			}
			if diff := cmp.Diff(tc.wantPackages, gotPackages); diff != "" {
				t.Errorf("unexpected packages:\n%s", diff)
			}
		})
	}
}
