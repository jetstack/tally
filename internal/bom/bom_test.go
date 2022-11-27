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
      "purl": "pkg:golang/foo/bar@v0.2.5",
      "externalReferences": [
        {
	  "type": "vcs",
	  "url": "https://github.com/foo/bar"
	}
      ]
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
    },
    {
      "bom-ref": "7",
      "type": "library",
      "name": "bar",
      "purl": "pkg:golang/github.com/foo/bar@v0.0.1",
      "externalReferences": [
        {
	  "type": "vcs",
	  "url": "git@github.com/foo/bar.git"
	}
      ]
    },
    {
      "bom-ref": "8",
      "type": "library",
      "name": "baz",
      "purl": "pkg:golang/github.com/foo/bar/v2/baz@v0.1.0"
    },
    {
      "bom-ref": "9",
      "type": "library",
      "name": "foo/bar",
      "purl": "pkg:golang/foo/bar@v0.2.5",
      "externalReferences": [
        {
	  "type": "vcs",
	  "url": "https://github.com/bar/foo"
	}
      ]
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
      <externalReferences>
        <reference type="vcs">
          <url>https://github.com/foo/bar</url>
        </reference>
      </externalReferences>
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
    <component type="library" bom-ref="7">
      <name>bar</name>
      <purl>pkg:golang/github.com/foo/bar@v0.0.1</purl>
      <externalReferences>
        <reference type="vcs">
          <url>git@github.com/foo/bar.git</url>
        </reference>
      </externalReferences>
    </component>
    <component type="library" bom-ref="8">
      <name>baz</name>
      <purl>pkg:golang/github.com/foo/bar/v2/baz@v0.1.0</purl>
    </component>
    <component type="library" bom-ref="9">
      <name>foo/bar</name>
      <purl>pkg:golang/foo/bar@v0.2.5</purl>
      <externalReferences>
        <reference type="vcs">
          <url>https://github.com/bar/foo</url>
        </reference>
      </externalReferences>
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
    },
    {
      "id": "8",
      "name": "bar",
      "purl": "pkg:golang/github.com/foo/bar@v0.0.1"
    },
    {
      "bom-ref": "9",
      "name": "baz",
      "purl": "pkg:golang/github.com/foo/bar/v2/baz@v0.1.0"
    }
  ]
}
`)

func TestPackagesFromBOM(t *testing.T) {
	testCases := map[string]struct {
		data     []byte
		format   Format
		wantPkgs types.Packages
		wantErr  bool
	}{
		"cyclonedx-json: detect expected packages": {
			data:   testCycloneDXJSON,
			format: FormatCycloneDXJSON,
			wantPkgs: types.Packages{
				{
					System: "GO",
					Name:   "foo/bar",
					Repositories: []string{
						"github.com/foo/bar",
						"github.com/bar/foo",
					},
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
				{
					System: "GO",
					Name:   "github.com/foo/bar",
					Repositories: []string{
						"github.com/foo/bar",
					},
				},
				{
					System: "GO",
					Name:   "github.com/foo/bar/v2/baz",
					Repositories: []string{
						"github.com/foo/bar",
					},
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
			wantPkgs: types.Packages{
				{
					System: "GO",
					Name:   "foo/bar",
					Repositories: []string{
						"github.com/foo/bar",
						"github.com/bar/foo",
					},
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
				{
					System: "GO",
					Name:   "github.com/foo/bar",
					Repositories: []string{
						"github.com/foo/bar",
					},
				},
				{
					System: "GO",
					Name:   "github.com/foo/bar/v2/baz",
					Repositories: []string{
						"github.com/foo/bar",
					},
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
			wantPkgs: types.Packages{
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
				{
					System: "GO",
					Name:   "github.com/foo/bar",
					Repositories: []string{
						"github.com/foo/bar",
					},
				},
				{
					System: "GO",
					Name:   "github.com/foo/bar/v2/baz",
					Repositories: []string{
						"github.com/foo/bar",
					},
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

func TestParseGithubURL(t *testing.T) {
	testCases := []struct {
		url      string
		wantRepo string
		wantErr  bool
	}{
		{
			url:      "https://github.com/foo/bar",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "https://github.com/foo/bar/tree/main/baz",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "https://github.com/foo/bar#baz",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "https://github.com/foo/bar/",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "https://github.com/foo/bar.git",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "https://github.com/foo/bar.git/",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "https://github.com/foo/bar.git?ref=baz",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "https://github.com/foo/bar.git?ref=baz",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "https://github.com/foo/bar?ref=something",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "https://github.com/foo/bar#something",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git://github.com/foo/bar",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git://github.com/foo/bar",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git://github.com/foo/bar/",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git://github.com/foo/bar.git",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git://github.com/foo/bar.git/",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git://github.com/foo/bar.git?ref=baz",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git://github.com/foo/bar?ref=something",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git://github.com/foo/bar#something",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git@github.com:foo/bar.git",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git@github.com:foo/bar.git",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git@github.com:foo/bar.git/",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git@github.com:foo/bar.git?ref=something",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git@github.com:foo/bar.git#something",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:     "https://github.com/foo",
			wantErr: true,
		},
		{
			url:     "https://gitlab.com/foo/bar",
			wantErr: true,
		},

		{
			url:     "git://gitlab.com/foo/bar.git",
			wantErr: true,
		},
		{
			url:     "git@gitlab.com:foo/bar.git",
			wantErr: true,
		},
		{
			url:     "github.com",
			wantErr: true,
		},
	}
	for _, tc := range testCases {
		gotRepo, err := parseGithubURL(tc.url)
		if err != nil && !tc.wantErr {
			t.Fatalf("unexpected error parsing %q: %s", tc.url, err)
		}
		if err == nil && tc.wantErr {
			t.Fatalf("expected error but got nil")
		}

		if gotRepo != tc.wantRepo {
			t.Fatalf("unexpected repo parsing %q; got %q but wanted %q", tc.url, gotRepo, tc.wantRepo)
		}
	}
}
