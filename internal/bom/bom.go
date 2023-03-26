package bom

import (
	"fmt"
	"io"

	"github.com/CycloneDX/cyclonedx-go"
	"github.com/jetstack/tally/internal/types"
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

// PackageRepositoriesFromBOM discovers packages and their associated
// repositories in an SBOM
func PackageRepositoriesFromBOM(r io.Reader, format Format) ([]*types.PackageRepositories, error) {
	switch format {
	case FormatCycloneDXJSON:
		bom, err := ParseCycloneDXBOM(r, cyclonedx.BOMFileFormatJSON)
		if err != nil {
			return nil, fmt.Errorf("parsing BOM in cyclonedx-json format: %w", err)
		}
		return PackageRepositoriesFromCycloneDXBOM(bom)
	case FormatCycloneDXXML:
		bom, err := ParseCycloneDXBOM(r, cyclonedx.BOMFileFormatXML)
		if err != nil {
			return nil, fmt.Errorf("parsing BOM in cyclonedx-xml format: %w", err)
		}
		return PackageRepositoriesFromCycloneDXBOM(bom)
	case FormatSyftJSON:
		bom, err := ParseSyftBOM(r)
		if err != nil {
			return nil, fmt.Errorf("parsing BOM in syft-json format: %w", err)
		}
		return PackageRepositoriesFromSyftBOM(bom)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}
