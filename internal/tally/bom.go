package tally

import (
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
)

// BOMFormats are all the supported SBOM formats
var BOMFormats = []BOMFormat{
	BOMFormatCycloneDXJSON,
	BOMFormatCycloneDXXML,
}

// PackagesFromBOM extracts packages from a supported SBOM format
func PackagesFromBOM(r io.Reader, format BOMFormat) ([]Package, error) {
	switch format {
	case BOMFormatCycloneDXJSON:
		return packagesFromCycloneDX(r, cyclonedx.BOMFileFormatJSON)
	case BOMFormatCycloneDXXML:
		return packagesFromCycloneDX(r, cyclonedx.BOMFileFormatXML)
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
	for _, component := range *cdxBOM.Components {
		if component.PackageURL == "" {
			continue
		}
		purl, err := packageurl.FromString(component.PackageURL)
		if err != nil {
			return nil, err
		}
		switch purl.Type {
		case packageurl.TypeNPM:
			name := purl.Name
			if purl.Namespace != "" {
				name = purl.Namespace + "/" + name
			}
			pkgs = append(
				pkgs,
				Package{
					System:  "NPM",
					Name:    name,
					Version: purl.Version,
				},
			)
		case packageurl.TypeGolang:
			pkgs = append(
				pkgs,
				Package{
					System:  "GO",
					Name:    strings.Join([]string{purl.Namespace, purl.Name}, "/"),
					Version: purl.Version,
				},
			)
		case packageurl.TypeMaven:
			pkgs = append(
				pkgs,
				Package{
					System:  "MAVEN",
					Name:    strings.Join([]string{purl.Namespace, purl.Name}, ":"),
					Version: purl.Version,
				},
			)
		case packageurl.TypePyPi:
			pkgs = append(
				pkgs,
				Package{
					System:  "PYPI",
					Name:    purl.Name,
					Version: purl.Version,
				},
			)
		case "cargo":
			pkgs = append(
				pkgs,
				Package{
					System:  "CARGO",
					Name:    purl.Name,
					Version: purl.Version,
				},
			)
		default:
		}

	}

	return pkgs, nil
}
