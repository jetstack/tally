package bom

import (
	"fmt"
	"io"

	"github.com/CycloneDX/cyclonedx-go"
	github_url "github.com/jetstack/tally/internal/github-url"
	"github.com/jetstack/tally/internal/types"
)

// ParseCycloneDXBOM parses a cyclonedx BOM in the specified format
func ParseCycloneDXBOM(r io.Reader, format cyclonedx.BOMFileFormat) (*cyclonedx.BOM, error) {
	bom := &cyclonedx.BOM{}
	if err := cyclonedx.NewBOMDecoder(r, format).Decode(bom); err != nil {
		return nil, fmt.Errorf("decoding cyclonedx BOM: %w", err)
	}

	return bom, nil
}

// PackageRepositoriesFromCycloneDXBOM extracts packages from a cyclonedx BOM
func PackageRepositoriesFromCycloneDXBOM(bom *cyclonedx.BOM) ([]*types.PackageRepositories, error) {
	var pkgRepos []*types.PackageRepositories
	if err := foreachComponentIn(
		bom,
		func(component cyclonedx.Component) error {
			pkgRepo, err := packageRepositoriesFromCycloneDXComponent(component)
			if err != nil {
				return err
			}
			if pkgRepo == nil {
				return nil
			}

			pkgRepos = appendPackageRepositories(pkgRepos, pkgRepo)

			return nil

		},
	); err != nil {
		return nil, fmt.Errorf("finding packages in BOM: %w", err)
	}

	return pkgRepos, nil
}

func packageRepositoriesFromCycloneDXComponent(component cyclonedx.Component) (*types.PackageRepositories, error) {
	if component.PackageURL == "" {
		return nil, nil
	}
	pkgRepo, err := packageRepositoriesFromPurl(component.PackageURL)
	if err != nil {
		return nil, err
	}
	if component.ExternalReferences == nil {
		return pkgRepo, nil
	}
	for _, ref := range *component.ExternalReferences {
		switch ref.Type {
		case cyclonedx.ERTypeVCS, cyclonedx.ERTypeDistribution, cyclonedx.ERTypeWebsite:
			repo, err := github_url.ToRepository(ref.URL)
			if err != nil {
				continue
			}
			pkgRepo.AddRepositories(repo)
		}
	}

	return pkgRepo, nil
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
