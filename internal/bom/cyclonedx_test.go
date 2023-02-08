package bom

import (
	"encoding/xml"
	"os"
	"testing"

	"github.com/CycloneDX/cyclonedx-go"
	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/types"
)

func TestCycloneDXParse(t *testing.T) {
	testCases := map[string]struct {
		path    string
		format  Format
		wantBOM *cyclonedx.BOM
		wantErr bool
	}{
		"json is parsed successfully": {
			path:   "testdata/cdx.json",
			format: FormatCycloneDXJSON,
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
			format:  FormatCycloneDXJSON,
			wantErr: true,
		},
		"xml": {
			path:   "testdata/cdx.xml",
			format: FormatCycloneDXXML,
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
			format:  FormatCycloneDXXML,
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

			bom, err := ParseBOM(r, tc.format)
			if err != nil && !tc.wantErr {
				t.Fatalf("unexpected error parsing BOM: %s", err)
			}
			if err == nil && tc.wantErr {
				t.Fatalf("expected parsing BOM but got nil")
			}

			if tc.wantErr {
				return
			}

			gotBOM := bom.(*cdxBOM).bom
			if diff := cmp.Diff(tc.wantBOM, gotBOM); diff != "" {
				t.Errorf("unexpected BOM:\n%s", diff)
			}
		})
	}
}

func TestCycloneDXBOMPackages(t *testing.T) {
	testCases := map[string]struct {
		bom          *cyclonedx.BOM
		wantPackages []types.Package
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
			wantPackages: []types.Package{
				{
					System: "GO",
					Name:   "foo/bar",
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
			wantPackages: []types.Package{
				{
					System: "GO",
					Name:   "foo/bar",
				},
				{
					System: "MAVEN",
					Name:   "org.hdrhistogram:HdrHistogram",
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
			wantPackages: []types.Package{
				{
					System: "GO",
					Name:   "foo/bar",
				},
				{
					System: "GO",
					Name:   "sigs.k8s.io/release-utils",
				},
				{
					System: "MAVEN",
					Name:   "org.hdrhistogram:HdrHistogram",
				},
				{
					System: "MAVEN",
					Name:   "com.github.package-url:packageurl-java",
				},
			},
		},
		"unsupported packages should be ignored": {
			bom: &cyclonedx.BOM{
				Components: &[]cyclonedx.Component{
					{
						PackageURL: "pkg:nuget/EnterpriseLibrary.Common@6.0.1304",
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
			wantPackages: []types.Package{
				{
					System: "MAVEN",
					Name:   "org.hdrhistogram:HdrHistogram",
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
			wantPackages: []types.Package{
				{
					System: "MAVEN",
					Name:   "org.hdrhistogram:HdrHistogram",
				},
				{
					System: "GO",
					Name:   "sigs.k8s.io/release-utils",
				},
				{
					System: "NPM",
					Name:   "zwitch",
				},
				{
					System: "CARGO",
					Name:   "getrandom",
				},
				{
					System: "PYPI",
					Name:   "zope.interface",
				},
			},
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			bom := &cdxBOM{
				bom: tc.bom,
			}
			gotPackages, err := bom.Packages()
			if err != nil {
				t.Fatalf("unexpected error getting packages from bom: %s", err)
			}
			if diff := cmp.Diff(tc.wantPackages, gotPackages); diff != "" {
				t.Errorf("unexpected packages:\n%s", diff)
			}
		})
	}
}

func TestCycloneDXBOMRepositories(t *testing.T) {
	testCases := map[string]struct {
		bom       *cyclonedx.BOM
		pkg       types.Package
		wantRepos []string
	}{
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
			pkg: types.Package{
				System: "GO",
				Name:   "foo/bar",
			},
			wantRepos: []string{
				"github.com/bar/foo",
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
			pkg: types.Package{
				System: "GO",
				Name:   "foo/bar",
			},
			wantRepos: []string{
				"github.com/bar/foo",
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
			pkg: types.Package{
				System: "GO",
				Name:   "foo/bar",
			},
			wantRepos: []string{
				"github.com/bar/foo",
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
			pkg: types.Package{
				System: "GO",
				Name:   "foo/bar",
			},
			wantRepos: []string{
				"github.com/bar/foo",
				"github.com/baz/bar",
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
			pkg: types.Package{
				System: "GO",
				Name:   "foo/bar",
			},
			wantRepos: []string{
				"github.com/bar/foo",
				"github.com/baz/bar",
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
			pkg: types.Package{
				System: "GO",
				Name:   "foo/bar",
			},
			wantRepos: []string{
				"github.com/bar/foo",
			},
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			bom := &cdxBOM{
				bom: tc.bom,
			}
			gotRepos, err := bom.Repositories(tc.pkg)
			if err != nil {
				t.Fatalf("unexpected error getting repositories from bom: %s", err)
			}
			if diff := cmp.Diff(tc.wantRepos, gotRepos); diff != "" {
				t.Errorf("unexpected packages:\n%s", diff)
			}
		})
	}
}
