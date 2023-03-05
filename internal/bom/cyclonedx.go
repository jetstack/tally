package bom

import (
	"fmt"
	"io"

	"github.com/CycloneDX/cyclonedx-go"
	github_url "github.com/jetstack/tally/internal/github-url"
	"github.com/jetstack/tally/internal/types"
)

type cdxBOM struct {
	bom *cyclonedx.BOM
}

func parseCycloneDX(r io.Reader, format cyclonedx.BOMFileFormat) (BOM, error) {
	bom := &cyclonedx.BOM{}
	if err := cyclonedx.NewBOMDecoder(r, format).Decode(bom); err != nil {
		return nil, fmt.Errorf("decoding cyclonedx BOM: %w", err)
	}

	return &cdxBOM{
		bom: bom,
	}, nil
}

// Packages returns all the supported packages in the BOM
func (b *cdxBOM) Packages() ([]types.Package, error) {
	var pkgs []types.Package
	if err := foreachComponentIn(
		b.bom,
		func(component cyclonedx.Component) error {
			pkg, err := packageFromCycloneDXComponent(component)
			if err != nil {
				return err
			}
			if pkg == nil {
				return nil
			}
			if containsPackage(pkgs, *pkg) {
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

// Repositories returns any repositories specified in the BOM for the provided
// package
func (b *cdxBOM) Repositories(pkg types.Package) ([]string, error) {
	var repos []string
	if err := foreachComponentIn(
		b.bom,
		func(component cyclonedx.Component) error {
			p, err := packageFromCycloneDXComponent(component)
			if err != nil {
				return err
			}
			if p == nil {
				return nil
			}
			if !p.Equals(pkg) {
				return nil
			}
			if component.ExternalReferences != nil {
				for _, ref := range *component.ExternalReferences {
					switch ref.Type {
					case cyclonedx.ERTypeVCS, cyclonedx.ERTypeDistribution, cyclonedx.ERTypeWebsite:
						repo, err := github_url.ToRepository(ref.URL)
						if err != nil {
							continue
						}
						if !contains(repos, repo) {
							repos = append(repos, repo)
						}
					}
				}
			}

			return nil

		},
	); err != nil {
		return nil, fmt.Errorf("finding repositories in BOM: %w", err)
	}

	return repos, nil
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

func packageFromCycloneDXComponent(component cyclonedx.Component) (*types.Package, error) {
	if component.PackageURL == "" {
		return nil, nil
	}
	pkg, err := packageFromPurl(component.PackageURL)
	if err != nil {
		return nil, err
	}

	return pkg, nil
}

func contains(vals []string, val string) bool {
	for _, v := range vals {
		if v == val {
			return true
		}
	}

	return false
}

func containsPackage(pkgs []types.Package, pkg types.Package) bool {
	for _, p := range pkgs {
		if p.Equals(pkg) {
			return true
		}
	}

	return false
}
