package bom

import (
	"encoding/xml"
	"os"
	"testing"

	"github.com/CycloneDX/cyclonedx-go"
	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/types"
)

func TestPackagesFromCycloneDXBOM(t *testing.T) {
	testCases := map[string]struct {
		path    string
		format  cyclonedx.BOMFileFormat
		wantBOM *cyclonedx.BOM
		wantErr bool
	}{
		"json is parsed successfully": {
			path:   "testdata/cdx.json",
			format: cyclonedx.BOMFileFormatJSON,
			wantBOM: &cyclonedx.BOM{
				BOMFormat:    "CycloneDX",
				SpecVersion:  cyclonedx.SpecVersion1_4,
				SerialNumber: "urn:uuid:5e0841b1-88e1-4dd8-b706-77457fb3e779",
				Version:      1,
				Metadata: &cyclonedx.Metadata{
					Component: &cyclonedx.Component{
						BOMRef:     "1234567",
						Type:       "application",
						Name:       "foo/bar",
						Version:    "v0.2.5",
						PackageURL: "pkg:golang/foo/bar@v0.2.5",
					},
				},
				Components: &[]cyclonedx.Component{
					{
						BOMRef:     "0",
						Type:       "library",
						Name:       "HdrHistogram",
						PackageURL: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9",
					},
					{
						BOMRef:     "1",
						Type:       "library",
						Name:       "adduser",
						PackageURL: "pkg:deb/debian/adduser@3.118?arch=all&distro=debian-11",
					},
				},
			},
		},
		"error is returned when parsing invalid json": {
			path:    "testdata/cdx.json.invalid",
			format:  cyclonedx.BOMFileFormatJSON,
			wantErr: true,
		},
		"xml": {
			path:   "testdata/cdx.xml",
			format: cyclonedx.BOMFileFormatXML,
			wantBOM: &cyclonedx.BOM{
				SpecVersion:  cyclonedx.SpecVersion1_3,
				SerialNumber: "urn:uuid:5e0841b1-88e1-4dd8-b706-77457fb3e779",
				Version:      1,
				XMLName: xml.Name{
					Space: "http://cyclonedx.org/schema/bom/1.3",
					Local: "bom",
				},
				XMLNS: "http://cyclonedx.org/schema/bom/1.3",
				Metadata: &cyclonedx.Metadata{
					Component: &cyclonedx.Component{
						BOMRef:     "1234567",
						Type:       "application",
						Name:       "foo/bar",
						Version:    "v0.2.5",
						PackageURL: "pkg:golang/foo/bar@v0.2.5",
					},
				},
				Components: &[]cyclonedx.Component{
					{
						BOMRef:     "0",
						Type:       "library",
						Name:       "HdrHistogram",
						PackageURL: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9",
					},
					{
						BOMRef:     "1",
						Type:       "library",
						Name:       "adduser",
						PackageURL: "pkg:deb/debian/adduser@3.118?arch=all&distro=debian-11",
					},
				},
			},
		},
		"error is returned when parsing invalid xml": {
			path:    "testdata/cdx.xml.invalid",
			format:  cyclonedx.BOMFileFormatXML,
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

			gotBOM, err := ParseCycloneDXBOM(r, tc.format)
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

func TestPackageRepositoriesFromCycloneDXBOM(t *testing.T) {
	testCases := map[string]struct {
		bom          *cyclonedx.BOM
		wantPackages []*types.PackageRepositories
	}{
		"an error should not be produced for an empty BOM": {
			bom: &cyclonedx.BOM{},
		},
		"an error should not be produced when metadata.component is nil": {
			bom: &cyclonedx.BOM{
				Metadata: &cyclonedx.Metadata{},
			},
		},
		"packages should be discovered in metadata.component": {
			bom: &cyclonedx.BOM{
				Metadata: &cyclonedx.Metadata{
					Component: &cyclonedx.Component{
						PackageURL: "pkg:golang/foo/bar@v0.2.5",
					},
				},
			},
			wantPackages: []*types.PackageRepositories{
				{
					Package: types.Package{
						Type: "golang",
						Name: "foo/bar",
					},
				},
			},
		},
		"packages should be discovered in metadata.component AND components": {
			bom: &cyclonedx.BOM{
				Metadata: &cyclonedx.Metadata{
					Component: &cyclonedx.Component{
						PackageURL: "pkg:golang/foo/bar@v0.2.5",
					}},
				Components: &[]cyclonedx.Component{
					{
						PackageURL: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9",
					},
					{
						PackageURL: "pkg:deb/debian/adduser@3.118?arch=all\u0026distro=debian-11",
					},
				},
			},
			wantPackages: []*types.PackageRepositories{
				{
					Package: types.Package{
						Type: "golang",
						Name: "foo/bar",
					},
				},
				{
					Package: types.Package{
						Type: "maven",
						Name: "org.hdrhistogram/HdrHistogram",
					},
				},
				{
					Package: types.Package{
						Type: "deb",
						Name: "debian/adduser",
					},
				},
			},
		},
		"packages should be discovered in nested components in metadata.component AND components": {
			bom: &cyclonedx.BOM{
				Metadata: &cyclonedx.Metadata{
					Component: &cyclonedx.Component{
						PackageURL: "pkg:golang/foo/bar@v0.2.5",
						Components: &[]cyclonedx.Component{
							{
								PackageURL: "pkg:golang/sigs.k8s.io/release-utils@v0.7.3",
							},
						},
					}},
				Components: &[]cyclonedx.Component{
					{
						PackageURL: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9",
						Components: &[]cyclonedx.Component{
							{
								Components: &[]cyclonedx.Component{
									{
										PackageURL: "pkg:maven/com.github.package-url/packageurl-java@1.4.1",
									},
								},
							},
						},
					},
				},
			},
			wantPackages: []*types.PackageRepositories{
				{
					Package: types.Package{
						Type: "golang",
						Name: "foo/bar",
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
						Type: "maven",
						Name: "org.hdrhistogram/HdrHistogram",
					},
				},
				{
					Package: types.Package{
						Type: "maven",
						Name: "com.github.package-url/packageurl-java",
					},
				},
			},
		},
		"components without a PackageURL should be ignored": {
			bom: &cyclonedx.BOM{
				Components: &[]cyclonedx.Component{
					{
						Name: "foo/bar",
					},
				},
			},
		},
		"duplicate packages should be ignored": {
			bom: &cyclonedx.BOM{
				Components: &[]cyclonedx.Component{
					{
						PackageURL: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.8",
					},
					{
						PackageURL: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9",
					},
					{
						PackageURL: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9",
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
		"all supported types should be discovered": {
			bom: &cyclonedx.BOM{
				Components: &[]cyclonedx.Component{
					{
						PackageURL: "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9",
					},
					{
						PackageURL: "pkg:golang/sigs.k8s.io/release-utils@v0.7.3",
					},
					{
						PackageURL: "pkg:npm/zwitch@2.0.2",
					},
					{
						PackageURL: "pkg:cargo/getrandom@0.2.7",
					},
					{
						PackageURL: "pkg:pypi/zope.interface@5.4.0",
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
		"repositories can be extracted from metadata.components": {
			bom: &cyclonedx.BOM{
				Components: &[]cyclonedx.Component{
					{
						PackageURL: "pkg:golang/foo/bar@v0.1.1",
						ExternalReferences: &[]cyclonedx.ExternalReference{
							{
								Type: cyclonedx.ERTypeVCS,
								URL:  "https://github.com/bar/foo",
							},
						},
					},
				},
			},
			wantPackages: []*types.PackageRepositories{
				{
					Package: types.Package{
						Type: "golang",
						Name: "foo/bar",
					},
					Repositories: []string{
						"github.com/bar/foo",
					},
				},
			},
		},
		"repositories can be extracted from components": {
			bom: &cyclonedx.BOM{
				Components: &[]cyclonedx.Component{
					{
						PackageURL: "pkg:golang/foo/bar@v0.1.1",
						ExternalReferences: &[]cyclonedx.ExternalReference{
							{
								Type: cyclonedx.ERTypeVCS,
								URL:  "https://github.com/bar/foo",
							},
						},
					},
				},
			},
			wantPackages: []*types.PackageRepositories{
				{
					Package: types.Package{
						Type: "golang",
						Name: "foo/bar",
					},
					Repositories: []string{
						"github.com/bar/foo",
					},
				},
			},
		},
		"repositories can be extracted from nested components": {
			bom: &cyclonedx.BOM{
				Components: &[]cyclonedx.Component{
					{
						PackageURL: "pkg:golang/bar/foo@v0.1.1",
						Components: &[]cyclonedx.Component{
							{
								PackageURL: "pkg:golang/foo/bar@v0.1.1",
								ExternalReferences: &[]cyclonedx.ExternalReference{
									{
										Type: cyclonedx.ERTypeVCS,
										URL:  "https://github.com/bar/foo",
									},
								},
							},
						},
					},
				},
			},
			wantPackages: []*types.PackageRepositories{
				{
					Package: types.Package{
						Type: "golang",
						Name: "bar/foo",
					},
				},
				{
					Package: types.Package{
						Type: "golang",
						Name: "foo/bar",
					},
					Repositories: []string{
						"github.com/bar/foo",
					},
				},
			},
		},
		"multiple repositories can be extracted from the same component": {
			bom: &cyclonedx.BOM{
				Components: &[]cyclonedx.Component{
					{
						PackageURL: "pkg:golang/foo/bar@v0.1.1",
						ExternalReferences: &[]cyclonedx.ExternalReference{
							{
								Type: cyclonedx.ERTypeVCS,
								URL:  "https://github.com/bar/foo",
							},
							{
								Type: cyclonedx.ERTypeVCS,
								URL:  "git@github.com:baz/bar",
							},
						},
					},
				},
			},
			wantPackages: []*types.PackageRepositories{
				{
					Package: types.Package{
						Type: "golang",
						Name: "foo/bar",
					},
					Repositories: []string{
						"github.com/bar/foo",
						"github.com/baz/bar",
					},
				},
			},
		},
		"multiple repositories can be extracted from different components": {
			bom: &cyclonedx.BOM{
				Metadata: &cyclonedx.Metadata{
					Component: &cyclonedx.Component{
						PackageURL: "pkg:golang/foo/bar@v0.1.1",
						ExternalReferences: &[]cyclonedx.ExternalReference{
							{
								Type: cyclonedx.ERTypeVCS,
								URL:  "https://github.com/bar/foo",
							},
						},
					},
				},
				Components: &[]cyclonedx.Component{
					{
						PackageURL: "pkg:golang/foo/bar@v0.1.1",
						ExternalReferences: &[]cyclonedx.ExternalReference{
							{
								Type: cyclonedx.ERTypeVCS,
								URL:  "git@github.com:baz/bar",
							},
						},
					},
				},
			},
			wantPackages: []*types.PackageRepositories{
				{
					Package: types.Package{
						Type: "golang",
						Name: "foo/bar",
					},
					Repositories: []string{
						"github.com/bar/foo",
						"github.com/baz/bar",
					},
				},
			},
		},
		"repositories are deduplicated": {
			bom: &cyclonedx.BOM{
				Metadata: &cyclonedx.Metadata{
					Component: &cyclonedx.Component{
						PackageURL: "pkg:golang/foo/bar@v0.1.1",
						ExternalReferences: &[]cyclonedx.ExternalReference{
							{
								Type: cyclonedx.ERTypeVCS,
								URL:  "https://github.com/bar/foo",
							},
						},
					},
				},
				Components: &[]cyclonedx.Component{
					{
						PackageURL: "pkg:golang/foo/bar@v0.1.1",
						ExternalReferences: &[]cyclonedx.ExternalReference{
							{
								Type: cyclonedx.ERTypeVCS,
								URL:  "https://github.com/bar/foo",
							},
						},
					},
				},
			},
			wantPackages: []*types.PackageRepositories{
				{
					Package: types.Package{
						Type: "golang",
						Name: "foo/bar",
					},
					Repositories: []string{
						"github.com/bar/foo",
					},
				},
			},
		},
		"repositories are extracted and deduplicated from all supported types": {
			bom: &cyclonedx.BOM{
				Components: &[]cyclonedx.Component{
					{
						PackageURL: "pkg:golang/foo/bar@v0.1.1",
						ExternalReferences: &[]cyclonedx.ExternalReference{
							{
								Type: cyclonedx.ERTypeVCS,
								URL:  "https://github.com/bar/foo",
							},
							{
								Type: cyclonedx.ERTypeDistribution,
								URL:  "https://github.com/bar/foo.git",
							},
							{
								Type: cyclonedx.ERTypeWebsite,
								URL:  "http://github.com/bar/foo.git",
							},
						},
					},
					{
						PackageURL: "pkg:golang/foo/bar@v0.2.2",
						ExternalReferences: &[]cyclonedx.ExternalReference{
							{
								Type: cyclonedx.ERTypeWebsite,
								URL:  "https://github.com/bar/foo.git",
							},
						},
					},
					{
						PackageURL: "pkg:golang/foo/bar@v0.2.2",
						ExternalReferences: &[]cyclonedx.ExternalReference{
							{
								Type: cyclonedx.ERTypeDistribution,
								URL:  "https://github.com/foo/baz.git",
							},
						},
					},
					{
						PackageURL: "pkg:golang/foo/bar@v0.1.1",
						ExternalReferences: &[]cyclonedx.ExternalReference{
							{
								Type: cyclonedx.ERTypeWebsite,
								URL:  "https://github.com/foo/bar",
							},
						},
					},
				},
			},
			wantPackages: []*types.PackageRepositories{
				{
					Package: types.Package{
						Type: "golang",
						Name: "foo/bar",
					},
					Repositories: []string{
						"github.com/bar/foo",
						"github.com/foo/baz",
						"github.com/foo/bar",
					},
				},
			},
		},
		"repository is extracted from the package url": {
			bom: &cyclonedx.BOM{
				Components: &[]cyclonedx.Component{
					{
						PackageURL: "pkg:pypi/foo.bar@5.4.0?vcs_url=git+git+ssh://git@github.com:foo/bar.git#v5.4.0",
					},
				},
			},
			wantPackages: []*types.PackageRepositories{
				{
					Package: types.Package{
						Type: "pypi",
						Name: "foo.bar",
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
			gotPackages, err := PackageRepositoriesFromCycloneDXBOM(tc.bom)
			if err != nil {
				t.Fatalf("unexpected error getting packages from bom: %s", err)
			}
			if diff := cmp.Diff(tc.wantPackages, gotPackages); diff != "" {
				t.Errorf("unexpected packages:\n%s", diff)
			}
		})
	}
}
