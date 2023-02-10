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

// BOM is a generic representation of a BOM
type BOM interface {
	// Packages retrieves all the supported packages from the BOM
	Packages() ([]types.Package, error)

	// Repositories retrieves any repository information from the BOM for
	// the provided package
	Repositories(types.Package) ([]string, error)
}

// ParseBOM parses a BOM from a given format
func ParseBOM(r io.Reader, format Format) (BOM, error) {
	switch format {
	case FormatCycloneDXJSON:
		return parseCycloneDX(r, cyclonedx.BOMFileFormatJSON)
	case FormatCycloneDXXML:
		return parseCycloneDX(r, cyclonedx.BOMFileFormatXML)
	case FormatSyftJSON:
		return parseSyftJSON(r)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}
