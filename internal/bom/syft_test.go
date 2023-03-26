package bom

import (
	"os"
	"testing"

	"github.com/anchore/syft/syft/formats/syftjson/model"
	"github.com/anchore/syft/syft/pkg"
	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/types"
)

func TestParseSyftBOM(t *testing.T) {
	testCases := map[string]struct {
		path    string
		wantBOM *model.Document
		wantErr bool
	}{
		"json is parsed successfully": {
			path: "testdata/syft.json",
			wantBOM: &model.Document{
				Artifacts: []model.Package{
					{
						PackageBasicData: model.PackageBasicData{
							ID:   "0",
							Name: "foo",
							PURL: "pkg:golang/foo/bar@v0.2.5",
						},
					},
					{
						PackageBasicData: model.PackageBasicData{
							ID:   "1",
							Name: "HdrHistogram",
							PURL: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9",
						},
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

func TestPackageRepositoriesFromSyftBOM(t *testing.T) {
	testCases := map[string]struct {
		bom          *model.Document
		wantPackages []*types.PackageRepositories
	}{
		"an error should not be produced for an empty BOM": {
			bom: &model.Document{},
		},
		"components without a PURL should be ignored": {
			bom: &model.Document{
				Artifacts: []model.Package{
					{
						PackageBasicData: model.PackageBasicData{
							Name: "foo",
						},
					},
				},
			},
		},
		"duplicate packages should be ignored": {
			bom: &model.Document{
				Artifacts: []model.Package{
					{

						PackageBasicData: model.PackageBasicData{
							PURL: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.8",
						},
					},
					{

						PackageBasicData: model.PackageBasicData{
							PURL: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9",
						},
					},
					{

						PackageBasicData: model.PackageBasicData{
							PURL: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9",
						},
					},
				},
			},
			wantPackages: []*types.PackageRepositories{
				{
					Package: types.Package{
						Type: "maven",
						Name: "org.hdrhistogram/HdrHistogram",
					},
				},
			},
		},
		"multiple types should be discovered": {
			bom: &model.Document{
				Artifacts: []model.Package{
					{

						PackageBasicData: model.PackageBasicData{
							PURL: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9",
						},
					},
					{

						PackageBasicData: model.PackageBasicData{
							PURL: "pkg:golang/sigs.k8s.io/release-utils@v0.7.3",
						},
					},
					{

						PackageBasicData: model.PackageBasicData{
							PURL: "pkg:npm/zwitch@2.0.2",
						},
					},
					{

						PackageBasicData: model.PackageBasicData{
							PURL: "pkg:cargo/getrandom@0.2.7",
						},
					},
					{

						PackageBasicData: model.PackageBasicData{
							PURL: "pkg:pypi/zope.interface@5.4.0",
						},
					},
				},
			},
			wantPackages: []*types.PackageRepositories{
				{
					Package: types.Package{
						Type: "maven",
						Name: "org.hdrhistogram/HdrHistogram",
					},
				},
				{
					Package: types.Package{
						Type: "golang",
						Name: "sigs.k8s.io/release-utils",
					},
				},
				{
					Package: types.Package{
						Type: "npm",
						Name: "zwitch",
					},
				},
				{
					Package: types.Package{
						Type: "cargo",
						Name: "getrandom",
					},
				},
				{
					Package: types.Package{
						Type: "pypi",
						Name: "zope.interface",
					},
				},
			},
		},
		"should discover repositories from supported package types": {
			bom: &model.Document{
				Artifacts: []model.Package{
					{

						PackageBasicData: model.PackageBasicData{
							PURL: "pkg:pub/foobar@3.3.0?hosted_url=pub.hosted.org",
						},
						PackageCustomData: model.PackageCustomData{
							MetadataType: pkg.DartPubMetadataType,
							Metadata: pkg.DartPubMetadata{
								VcsURL: "github.com/foo/bar",
							},
						},
					},
					{

						PackageBasicData: model.PackageBasicData{
							PURL: "pkg:gem/foobar@2.1.4",
						},
						PackageCustomData: model.PackageCustomData{
							MetadataType: pkg.GemMetadataType,
							Metadata: pkg.GemMetadata{
								Homepage: "https://github.com/foo/bar",
							},
						},
					},
					{

						PackageBasicData: model.PackageBasicData{
							PURL: "pkg:composer/foo/bar@1.0.2",
						},
						PackageCustomData: model.PackageCustomData{
							MetadataType: pkg.PhpComposerJSONMetadataType,
							Metadata: pkg.PhpComposerJSONMetadata{
								Source: pkg.PhpComposerExternalReference{
									Type:      "git",
									URL:       "https://github.com/foo/bar.git",
									Reference: "6d9a552f0206a1db7feb442824540aa6c55e5b27",
								},
							},
						},
					},
					{

						PackageBasicData: model.PackageBasicData{
							PURL: "pkg:npm/foobar@6.14.6",
						},
						PackageCustomData: model.PackageCustomData{
							MetadataType: pkg.NpmPackageJSONMetadataType,
							Metadata: pkg.NpmPackageJSONMetadata{
								Homepage: "https://github.com/foo/bar",
							},
						},
					},
					{

						PackageBasicData: model.PackageBasicData{
							PURL: "pkg:npm/foobar1@6.14.6",
						},
						PackageCustomData: model.PackageCustomData{
							MetadataType: pkg.NpmPackageJSONMetadataType,
							Metadata: pkg.NpmPackageJSONMetadata{
								Homepage: "https://docs.npmjs.com/",
								URL:      "https://github.com/foo/bar1",
							},
						},
					},
					{

						PackageBasicData: model.PackageBasicData{
							PURL: "pkg:npm/barfoo@6.14.6",
						},
						PackageCustomData: model.PackageCustomData{
							MetadataType: pkg.NpmPackageJSONMetadataType,
							Metadata: pkg.NpmPackageJSONMetadata{
								Homepage: "https://github.com/bar/foo",
								URL:      "https://github.com/foo/bar",
							},
						},
					},
					{

						PackageBasicData: model.PackageBasicData{
							PURL: "pkg:pypi/foobar@v0.1.0",
						},
						PackageCustomData: model.PackageCustomData{
							MetadataType: pkg.PythonPackageMetadataType,
							Metadata: pkg.PythonPackageMetadata{
								DirectURLOrigin: &pkg.PythonDirectURLOriginInfo{
									VCS:      "git",
									URL:      "https://github.com/foo/bar.git",
									CommitID: "6d9a552f0206a1db7feb442824540aa6c55e5b27",
								},
							},
						},
					},
				},
			},
			wantPackages: []*types.PackageRepositories{
				{
					Package: types.Package{
						Type: "pub",
						Name: "foobar",
					},
					Repositories: []string{
						"github.com/foo/bar",
					},
				},
				{
					Package: types.Package{
						Type: "gem",
						Name: "foobar",
					},
					Repositories: []string{
						"github.com/foo/bar",
					},
				},
				{
					Package: types.Package{
						Type: "composer",
						Name: "foo/bar",
					},
					Repositories: []string{
						"github.com/foo/bar",
					},
				},
				{
					Package: types.Package{
						Type: "npm",
						Name: "foobar",
					},
					Repositories: []string{
						"github.com/foo/bar",
					},
				},
				{
					Package: types.Package{
						Type: "npm",
						Name: "foobar1",
					},
					Repositories: []string{
						"github.com/foo/bar1",
					},
				},
				{
					Package: types.Package{
						Type: "npm",
						Name: "barfoo",
					},
					Repositories: []string{
						"github.com/bar/foo",
						"github.com/foo/bar",
					},
				},
				{
					Package: types.Package{
						Type: "pypi",
						Name: "foobar",
					},
					Repositories: []string{
						"github.com/foo/bar",
					},
				},
			},
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {

			gotPackages, err := PackageRepositoriesFromSyftBOM(tc.bom)
			if err != nil {
				t.Fatalf("unexpected error getting packages from bom: %s", err)
			}
			if diff := cmp.Diff(tc.wantPackages, gotPackages); diff != "" {
				t.Errorf("unexpected packages:\n%s", diff)
			}
		})
	}
}
