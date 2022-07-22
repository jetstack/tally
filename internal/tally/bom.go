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
func PackagesFromBOM(r io.Reader, format BOMFormat) ([]Package, error) {
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

func packagesFromCycloneDX(r io.Reader, format cyclonedx.BOMFileFormat) ([]Package, error) {
	cdxBOM := &cyclonedx.BOM{}
	if err := cyclonedx.NewBOMDecoder(r, format).Decode(cdxBOM); err != nil {
		return nil, fmt.Errorf("decoding cyclonedx BOM: %w", err)
	}
	var pkgs []Package
	if cdxBOM.Components == nil {
		return pkgs, nil
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

func packagesFromSyftJSON(r io.Reader) ([]Package, error) {
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

	return pkgs, nil
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
			Type:    PackageTypeNPM,
			Name:    name,
			Version: purl.Version,
		}, nil
	case packageurl.TypeGolang:
		name := purl.Name
		if purl.Namespace != "" {
			name = purl.Namespace + "/" + purl.Name
		}
		return Package{
			Type:    PackageTypeGo,
			Name:    name,
			Version: purl.Version,
		}, nil
	case packageurl.TypeMaven:
		return Package{
			Type:    PackageTypeMaven,
			Name:    strings.Join([]string{purl.Namespace, purl.Name}, ":"),
			Version: purl.Version,
		}, nil
	case packageurl.TypePyPi:
		return Package{
			Type:    PackageTypePyPI,
			Name:    purl.Name,
			Version: purl.Version,
		}, nil
	case "cargo":
		return Package{
			Type:    PackageTypeCargo,
			Name:    purl.Name,
			Version: purl.Version,
		}, nil
	default:
		return Package{}, ErrUnsupportedPackageType
	}
}
