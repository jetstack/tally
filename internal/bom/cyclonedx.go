package bom

import (
	"fmt"
	"io"

	"github.com/CycloneDX/cyclonedx-go"
	github_url "github.com/jetstack/tally/internal/github-url"
)

// ParseCycloneDXBOM parses a cyclonedx BOM in the specified format
func ParseCycloneDXBOM(r io.Reader, format cyclonedx.BOMFileFormat) (*cyclonedx.BOM, error) {
	bom := &cyclonedx.BOM{}
	if err := cyclonedx.NewBOMDecoder(r, format).Decode(bom); err != nil {
		return nil, fmt.Errorf("decoding cyclonedx BOM: %w", err)
	}

	return bom, nil
}

// PackagesFromCycloneDXBOM extracts packages from a cyclonedx BOM
func PackagesFromCycloneDXBOM(bom *cyclonedx.BOM) ([]Package, error) {
	var pkgs []Package
	if err := foreachComponentIn(
		bom,
		func(component cyclonedx.Component) error {
			pkg, err := packageFromCycloneDXComponent(component)
			if err != nil {
				return err
			}
			if pkg == nil {
				return nil
			}
			for i, p := range pkgs {
				if p.Name != pkg.Name {
					continue
				}
				if p.Type != pkg.Type {
					continue
				}
				for _, repo := range pkg.Repositories {
					if contains(p.Repositories, repo) {
						continue
					}
					pkgs[i].Repositories = append(pkgs[i].Repositories, repo)
				}

				return nil
			}

			pkgs = append(pkgs, *pkg)

			return nil

		},
	); err != nil {
		return nil, fmt.Errorf("finding packages in BOM: %w", err)
	}

	return pkgs, nil
}

func packageFromCycloneDXComponent(component cyclonedx.Component) (*Package, error) {
	if component.PackageURL == "" {
		return nil, nil
	}
	pkg, err := packageFromPurl(component.PackageURL)
	if err != nil {
		return nil, err
	}
	if component.ExternalReferences == nil {
		return pkg, nil
	}
	for _, ref := range *component.ExternalReferences {
		switch ref.Type {
		case cyclonedx.ERTypeVCS, cyclonedx.ERTypeDistribution, cyclonedx.ERTypeWebsite:
			repo, err := github_url.ToRepository(ref.URL)
			if err != nil {
				continue
			}
			if !contains(pkg.Repositories, repo) {
				pkg.Repositories = append(pkg.Repositories, repo)
			}
		}
	}

	return pkg, nil
}

func foreachComponentIn(bom *cyclonedx.BOM, fn func(component cyclonedx.Component) error) error {
	var components []cyclonedx.Component
	if bom.Metadata != nil && bom.Metadata.Component != nil {
		components = append(components, *bom.Metadata.Component)
	}
	if bom.Components != nil {
		components = append(components, *bom.Components...)
	}
	return walkCycloneDXComponents(components, fn)
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

func contains(vals []string, val string) bool {
	for _, v := range vals {
		if v == val {
			return true
		}
	}

	return false
}
