package bom

import (
	"fmt"
	"io"

	"github.com/CycloneDX/cyclonedx-go"
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

// Package is a generic representation of a package in an SBOM
type Package struct {
	Type         string
	Name         string
	Repositories []string
}

// PackagesFromBOM discovers packages in an SBOM
func PackagesFromBOM(r io.Reader, format Format) ([]Package, error) {
	switch format {
	case FormatCycloneDXJSON:
		bom, err := ParseCycloneDXBOM(r, cyclonedx.BOMFileFormatJSON)
		if err != nil {
			return nil, fmt.Errorf("parsing BOM in cyclonedx-json format: %w", err)
		}
		return PackagesFromCycloneDXBOM(bom)
	case FormatCycloneDXXML:
		bom, err := ParseCycloneDXBOM(r, cyclonedx.BOMFileFormatXML)
		if err != nil {
			return nil, fmt.Errorf("parsing BOM in cyclonedx-xml format: %w", err)
		}
		return PackagesFromCycloneDXBOM(bom)
	case FormatSyftJSON:
		bom, err := ParseSyftBOM(r)
		if err != nil {
			return nil, fmt.Errorf("parsing BOM in syft-json format: %w", err)
		}
		return PackagesFromSyftBOM(bom)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}
