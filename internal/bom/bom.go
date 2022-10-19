package bom

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/CycloneDX/cyclonedx-go"
	"github.com/jetstack/tally/internal/types"
	"github.com/package-url/packageurl-go"
)

// Format is a supported SBOM format
type Format string

const (
	FormatCycloneDXJSON Format = "cyclonedx-json"
	FormatCycloneDXXML  Format = "cyclonedx-xml"
	FormatSyftJSON      Format = "syft-json"
)

// Formats are all the supported SBOM formats
var Formats = []Format{
	FormatCycloneDXJSON,
	FormatCycloneDXXML,
	FormatSyftJSON,
}

var (
	ErrUnsupportedPackageType = errors.New("unsupported package type")
)

// PackagesFromBOM extracts packages from a supported SBOM format
func PackagesFromBOM(r io.Reader, format Format) ([]types.Package, error) {
	switch format {
	case FormatCycloneDXJSON:
		return packagesFromCycloneDX(r, cyclonedx.BOMFileFormatJSON)
	case FormatCycloneDXXML:
		return packagesFromCycloneDX(r, cyclonedx.BOMFileFormatXML)
	case FormatSyftJSON:
		return packagesFromSyftJSON(r)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func packagesFromCycloneDX(r io.Reader, format cyclonedx.BOMFileFormat) ([]types.Package, error) {
	cdxBOM := &cyclonedx.BOM{}
	if err := cyclonedx.NewBOMDecoder(r, format).Decode(cdxBOM); err != nil {
		return nil, fmt.Errorf("decoding cyclonedx BOM: %w", err)
	}
	var (
		pkgs       []types.Package
		components []cyclonedx.Component
	)
	if cdxBOM.Metadata != nil && cdxBOM.Metadata.Component != nil {
		components = append(components, *cdxBOM.Metadata.Component)
	}
	if cdxBOM.Components != nil {
		components = append(components, *cdxBOM.Components...)
	}
	if err := walkCycloneDXComponents(components, func(component cyclonedx.Component) error {
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

	return pkgs, nil
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

func packagesFromSyftJSON(r io.Reader) ([]types.Package, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	doc := &syftJSON{}
	if err := json.Unmarshal(data, doc); err != nil {
		return nil, err
	}
	var pkgs []types.Package
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

	return pkgs, nil
}

func packageFromPurl(purl packageurl.PackageURL) (types.Package, error) {
	switch purl.Type {
	case packageurl.TypeNPM:
		name := purl.Name
		if purl.Namespace != "" {
			name = purl.Namespace + "/" + name
		}
		return types.Package{
			System: "NPM",
			Name:   name,
		}, nil
	case packageurl.TypeGolang:
		name := purl.Name
		if purl.Namespace != "" {
			name = purl.Namespace + "/" + purl.Name
		}
		return types.Package{
			System: "GO",
			Name:   name,
		}, nil
	case packageurl.TypeMaven:
		return types.Package{
			System: "MAVEN",
			Name:   strings.Join([]string{purl.Namespace, purl.Name}, ":"),
		}, nil
	case packageurl.TypePyPi:
		return types.Package{
			System: "PYPI",
			Name:   purl.Name,
		}, nil
	case "cargo":
		return types.Package{
			System: "CARGO",
			Name:   purl.Name,
		}, nil
	default:
		return types.Package{}, ErrUnsupportedPackageType
	}
}

func containsPackage(pkgs []types.Package, pkg types.Package) bool {
	for _, p := range pkgs {
		if p.Equals(pkg) {
			return true
		}
	}

	return false
}
