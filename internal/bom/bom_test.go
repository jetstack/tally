package bom

import (
	"bytes"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/types"
)

var testCycloneDXJSON = []byte(`
{
  "bomFormat": "CycloneDX",
  "specVersion": "1.4",
  "serialNumber": "urn:uuid:5e0841b1-88e1-4dd8-b706-77457fb3e779",
  "version": 1,
  "metadata": {
    "component": {
      "bom-ref": "1234567",
      "type": "application",
      "name": "foo/bar",
      "version": "v0.2.5",
      "purl": "pkg:golang/foo/bar@v0.2.5"
    }
  },
  "components": [
    {
      "bom-ref": "0",
      "type": "library",
      "name": "HdrHistogram",
      "purl": "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9"
    },
    {
      "bom-ref": "1",
      "type": "library",
      "name": "adduser",
      "purl": "pkg:deb/debian/adduser@3.118?arch=all\u0026distro=debian-11"
    },
    {
      "bom-ref": "2",
      "type": "library",
      "name": "release-utils",
      "purl": "pkg:golang/sigs.k8s.io/release-utils@v0.7.3"
    },
    {
      "bom-ref": "3",
      "type": "library",
      "name": "zwitch",
      "purl": "pkg:npm/zwitch@2.0.2"
    },
    {
      "bom-ref": "4",
      "type": "library",
      "name": "getrandom",
      "purl": "pkg:cargo/getrandom@0.2.7"
    },
    {
      "bom-ref": "5",
      "type": "library",
      "name": "barfoo"
    },
    {
      "bom-ref": "6",
      "type": "library",
      "name": "zope.interface",
      "purl": "pkg:pypi/zope.interface@5.4.0"
    }
  ]
}
`)

var testCycloneDXXML = []byte(`
<?xml version="1.0" encoding="utf-8"?>
<bom xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" serialNumber="urn:uuid:5e0841b1-88e1-4dd8-b706-77457fb3e779" version="1" xmlns="http://cyclonedx.org/schema/bom/1.3">
  <metadata>
    <component type="application" bom-ref="1234567">
      <name>foo/bar</name>
      <version>v0.2.5</version>
      <purl>pkg:golang/foo/bar@v0.2.5</purl>
    </component>
  </metadata>
  <components>
    <component type="library" bom-ref="0">
      <name>HdrHistogram</name>
      <purl>pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9</purl>
    </component>
    <component type="library" bom-ref="1">
      <name>adduser</name>
      <purl>pkg:deb/debian/adduser@3.118?arch=all&amp;distro=debian-11</purl>
    </component>
    <component type="library" bom-ref="2">
      <name>release-utils</name>
      <purl>pkg:golang/sigs.k8s.io/release-utils@v0.7.3</purl>
    </component>
    <component type="library" bom-ref="3">
      <name>zwitch</name>
      <purl>pkg:npm/zwitch@2.0.2</purl>
    </component>
    <component type="library" bom-ref="4">
      <name>getrandom</name>
      <purl>pkg:cargo/getrandom@0.2.7</purl>
    </component>
    <component type="library" bom-ref="5">
      <name>barfoo</name>
    </component>
    <component type="library" bom-ref="6">
      <name>zope.interface</name>
      <purl>pkg:pypi/zope.interface@5.4.0</purl>
    </component>
  </components>
</bom>
`)

var testSyftJSON = []byte(`
{
  "artifacts": [
    {
      "id": "0",
      "name": "foo",
      "purl": "pkg:golang/foo/bar@v0.2.5"
    },
    {
      "id": "1",
      "name": "HdrHistogram",
      "purl": "pkg:maven/org.hdrhistogram/HdrHistogram@2.1.9"
    },
    {
      "id": "2",
      "name": "adduser",
      "purl": "pkg:deb/debian/adduser@3.118?arch=all\u0026distro=debian-11"
    },
    {
      "id": "3",
      "name": "release-utils",
      "purl": "pkg:golang/sigs.k8s.io/release-utils@v0.7.3"
    },
    {
      "id": "4",
      "name": "zwitch",
      "purl": "pkg:npm/zwitch@2.0.2"
    },
    {
      "id": "5",
      "name": "getrandom",
      "purl": "pkg:cargo/getrandom@0.2.7"
    },
    {
      "id": "6",
      "name": "barfoo"
    },
    {
      "id": "7",
      "name": "zope.interface",
      "purl": "pkg:pypi/zope.interface@5.4.0"
    }
  ]
}
`)

func TestPackagesFromBOM(t *testing.T) {
	testCases := map[string]struct {
		data     []byte
		format   Format
		wantPkgs []types.Package
		wantErr  bool
	}{
		"cyclonedx-json: detect expected packages": {
			data:   testCycloneDXJSON,
			format: FormatCycloneDXJSON,
			wantPkgs: []types.Package{
				{
					System: "GO",
					Name:   "foo/bar",
				},
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
		"cyclonedx-json: returns error when input is cyclonedx xml": {
			data:    testCycloneDXXML,
			format:  FormatCycloneDXJSON,
			wantErr: true,
		},
		"cyclonedx-xml: detect expected packages": {
			data:   testCycloneDXXML,
			format: FormatCycloneDXXML,
			wantPkgs: []types.Package{
				{
					System: "GO",
					Name:   "foo/bar",
				},
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
		"cyclonedx-xml: returns error when input is cyclonedx json": {
			data:    testCycloneDXJSON,
			format:  FormatCycloneDXXML,
			wantErr: true,
		},
		"syft-json: detect expected packages": {
			data:   testSyftJSON,
			format: FormatSyftJSON,
			wantPkgs: []types.Package{
				{
					System: "GO",
					Name:   "foo/bar",
				},
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
		"syft-json: returns error when input is not json": {
			data:    []byte(`foobar`),
			format:  FormatSyftJSON,
			wantErr: true,
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			gotPkgs, err := PackagesFromBOM(bytes.NewReader(tc.data), tc.format)
			if err != nil && !tc.wantErr {
				t.Fatalf("unexpected error: %s", err)
			}
			if err == nil && tc.wantErr {
				t.Fatalf("expected error but got nil")
			}

			if diff := cmp.Diff(gotPkgs, tc.wantPkgs); diff != "" {
				t.Fatalf("unexpected packages:\n%s", diff)
			}
		})
	}
}
