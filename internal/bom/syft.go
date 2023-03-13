package bom

import (
	"encoding/json"
	"io"
)

type syftJSON struct {
	Artifacts []syftArtifact `json:"artifacts"`
}

type syftArtifact struct {
	Purl string `json:"purl"`
}

// ParseSyftBOM parses a syft SBOM
func ParseSyftBOM(r io.Reader) (*syftJSON, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	doc := &syftJSON{}
	if err := json.Unmarshal(data, doc); err != nil {
		return nil, err
	}

	return doc, nil
}

// PackagesFromSyftBOM discovers packages in a Syft BOM
func PackagesFromSyftBOM(doc *syftJSON) ([]Package, error) {
	var pkgs []Package
	for _, a := range doc.Artifacts {
		if a.Purl == "" {
			continue
		}
		pkg, err := packageFromPurl(a.Purl)
		if err != nil {
			return nil, err
		}
		if containsPackage(pkgs, *pkg) {
			continue
		}
		pkgs = append(pkgs, *pkg)
	}
	return pkgs, nil
}

func containsPackage(pkgs []Package, pkg Package) bool {
	for _, p := range pkgs {
		if p.Name != pkg.Name {
			continue
		}
		if p.Type != pkg.Type {
			continue
		}

		return true
	}

	return false
}
