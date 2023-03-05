package bom

import (
	"encoding/json"
	"io"

	"github.com/jetstack/tally/internal/types"
)

type syftJSON struct {
	Artifacts []syftArtifact `json:"artifacts"`
}

type syftArtifact struct {
	Purl string `json:"purl"`
}

type syftBOM struct {
	bom *syftJSON
}

func parseSyftJSON(r io.Reader) (BOM, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	doc := &syftJSON{}
	if err := json.Unmarshal(data, doc); err != nil {
		return nil, err
	}
	return &syftBOM{
		bom: doc,
	}, nil
}

// Packages returns all the supported packages in the BOM
func (bom *syftBOM) Packages() ([]types.Package, error) {
	var pkgs []types.Package
	for _, a := range bom.bom.Artifacts {
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

// Repositories doesn't return anything for the syft-json format
func (bom *syftBOM) Repositories(pkg types.Package) ([]string, error) {
	return []string{}, nil
}
