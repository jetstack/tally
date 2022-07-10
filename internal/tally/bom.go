package tally

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/CycloneDX/cyclonedx-go"
	"github.com/package-url/packageurl-go"
)

// BOMFormat is a supported SBOM format
type BOMFormat string

const (
	BOMFormatCycloneDXJSON BOMFormat = "cyclonedx-json"
	BOMFormatCycloneDXXML  BOMFormat = "cyclonedx-xml"
	BOMFormatSyftJSON      BOMFormat = "syft-json"
)

// BOMFormats are all the supported SBOM formats
var BOMFormats = []BOMFormat{
	BOMFormatCycloneDXJSON,
	BOMFormatCycloneDXXML,
	BOMFormatSyftJSON,
}

// PackagesFromBOM extracts packages from a supported SBOM format
func PackagesFromBOM(r io.Reader, format BOMFormat) (Packages, error) {
	switch format {
	case BOMFormatCycloneDXJSON:
		return packagesFromCycloneDX(r, cyclonedx.BOMFileFormatJSON)
	case BOMFormatCycloneDXXML:
		return packagesFromCycloneDX(r, cyclonedx.BOMFileFormatXML)
	case BOMFormatSyftJSON:
		return packagesFromSyftJSON(r)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func packagesFromCycloneDX(r io.Reader, format cyclonedx.BOMFileFormat) (Packages, error) {
	cdxBOM := &cyclonedx.BOM{}
	if err := cyclonedx.NewBOMDecoder(r, format).Decode(cdxBOM); err != nil {
		return nil, fmt.Errorf("decoding cyclonedx BOM: %w", err)
	}
	var pkgs []Package
	if cdxBOM.Components == nil {
		return &packages{pkgs}, nil
	}
	if err := walkCycloneDXComponents(*cdxBOM.Components, func(component cyclonedx.Component) error {
		if component.PackageURL == "" {
			return nil
		}
		purl, err := packageurl.FromString(component.PackageURL)
		if err != nil {
			return err
		}
		pkg, err := packageFromPurl(purl)
		if errors.Is(err, ErrUnsupportedPackageType) {
			return nil
		}
		if err != nil {
			return err
		}
		if !containsPackage(pkgs, pkg) {
			pkgs = append(pkgs, pkg)
		}

		return nil

	}); err != nil {
		return nil, err
	}

	return &packages{pkgs}, nil
}

func walkCycloneDXComponents(components []cyclonedx.Component, fn func(cyclonedx.Component) error) error {
	for _, component := range components {
		if err := fn(component); err != nil {
			return err
		}
		if component.Components == nil {
			continue
		}
		if err := walkCycloneDXComponents(*component.Components, fn); err != nil {
			return err
		}
	}

	return nil
}

type syftJSON struct {
	Artifacts []syftArtifact `json:"artifacts"`
}

type syftArtifact struct {
	Purl string `json:"purl"`
}

func packagesFromSyftJSON(r io.Reader) (Packages, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	doc := &syftJSON{}
	if err := json.Unmarshal(data, doc); err != nil {
		return nil, err
	}
	var pkgs []Package
	for _, a := range doc.Artifacts {
		if a.Purl == "" {
			continue
		}
		purl, err := packageurl.FromString(a.Purl)
		if err != nil {
			return nil, err
		}
		pkg, err := packageFromPurl(purl)
		if errors.Is(err, ErrUnsupportedPackageType) {
			continue
		}
		if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, pkg)
	}

	return &packages{pkgs}, nil
}

var ErrUnsupportedPackageType = errors.New("unsupported package type")

func packageFromPurl(purl packageurl.PackageURL) (Package, error) {
	switch purl.Type {
	case packageurl.TypeNPM:
		name := purl.Name
		if purl.Namespace != "" {
			name = purl.Namespace + "/" + name
		}
		return Package{
			System:  "NPM",
			Name:    name,
			Version: purl.Version,
		}, nil
	case packageurl.TypeGolang:
		name := purl.Name
		if purl.Namespace != "" {
			name = purl.Namespace + "/" + purl.Name
		}
		var repo string
		if strings.HasPrefix(name, "github.com/") {
			c := strings.Split(name, "/")
			if len(c) >= 3 {
				repo = strings.Join([]string{c[0], c[1], c[2]}, "/")
			}
		}
		return Package{
			System:         "GO",
			Name:           name,
			Version:        purl.Version,
			RepositoryName: repo,
		}, nil
	case packageurl.TypeMaven:
		return Package{
			System:  "MAVEN",
			Name:    strings.Join([]string{purl.Namespace, purl.Name}, ":"),
			Version: purl.Version,
		}, nil
	case packageurl.TypePyPi:
		return Package{
			System:  "PYPI",
			Name:    purl.Name,
			Version: purl.Version,
		}, nil
	case "cargo":
		return Package{
			System:  "CARGO",
			Name:    purl.Name,
			Version: purl.Version,
		}, nil
	default:
		return Package{}, ErrUnsupportedPackageType
	}
}
