package bom

import (
	"encoding/json"
	"io"

	"github.com/anchore/syft/syft/formats/syftjson/model"
	syft "github.com/anchore/syft/syft/pkg"
	github_url "github.com/jetstack/tally/internal/github-url"
	"github.com/jetstack/tally/internal/types"
)

// ParseSyftBOM parses a syft SBOM
func ParseSyftBOM(r io.Reader) (*model.Document, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	doc := &model.Document{}
	if err := json.Unmarshal(data, doc); err != nil {
		return nil, err
	}

	return doc, nil
}

// PackageRepositoriesFromSyftBOM discovers packages in a Syft BOM
func PackageRepositoriesFromSyftBOM(doc *model.Document) ([]*types.PackageRepositories, error) {
	var pkgRepos []*types.PackageRepositories
	for _, a := range doc.Artifacts {
		pkgRepo, err := packageRepositoriesFromSyftPackage(a)
		if err != nil {
			return nil, err
		}
		if pkgRepo == nil {
			continue
		}

		pkgRepos = appendPackageRepositories(pkgRepos, pkgRepo)
	}

	return pkgRepos, nil
}

func packageRepositoriesFromSyftPackage(pkg model.Package) (*types.PackageRepositories, error) {
	if pkg.PURL == "" {
		return nil, nil
	}

	pkgRepo, err := packageRepositoriesFromPurl(pkg.PURL)
	if err != nil {
		return nil, err
	}
	if pkgRepo == nil {
		return nil, nil
	}

	pkgRepo.AddRepositories(repositoriesFromSyftPackage(pkg)...)

	return pkgRepo, nil
}

func repositoriesFromSyftPackage(pkg model.Package) []string {
	var repos []string
	switch pkg.MetadataType {
	case syft.DartPubMetadataType:
		metadata, ok := pkg.Metadata.(syft.DartPubMetadata)
		if ok {
			repo, err := github_url.ToRepository(metadata.VcsURL)
			if err == nil {
				repos = append(repos, repo)
			}
		}
	case syft.GemMetadataType:
		metadata, ok := pkg.Metadata.(syft.GemMetadata)
		if ok {
			repo, err := github_url.ToRepository(metadata.Homepage)
			if err == nil {
				repos = append(repos, repo)
			}
		}
	case syft.PhpComposerJSONMetadataType:
		metadata, ok := pkg.Metadata.(syft.PhpComposerJSONMetadata)
		if ok {
			repo, err := github_url.ToRepository(metadata.Source.URL)
			if err == nil {
				repos = append(repos, repo)
			}
		}
	case syft.NpmPackageJSONMetadataType:
		metadata, ok := pkg.Metadata.(syft.NpmPackageJSONMetadata)
		if ok {
			repo, err := github_url.ToRepository(metadata.Homepage)
			if err == nil {
				repos = append(repos, repo)
			}
			repo, err = github_url.ToRepository(metadata.URL)
			if err == nil {
				repos = append(repos, repo)
			}
		}
	case syft.PythonPackageMetadataType:
		metadata, ok := pkg.Metadata.(syft.PythonPackageMetadata)
		if ok {
			if metadata.DirectURLOrigin != nil {
				repo, err := github_url.ToRepository(metadata.DirectURLOrigin.URL)
				if err == nil {
					repos = append(repos, repo)
				}
			}
		}

	}

	return repos
}

func appendPackageRepositories(pkgRepos []*types.PackageRepositories, pkgRepo *types.PackageRepositories) []*types.PackageRepositories {
	for _, p := range pkgRepos {
		if !p.Equals(pkgRepo.Package) {
			continue
		}

		p.AddRepositories(pkgRepo.Repositories...)

		return pkgRepos
	}

	return append(pkgRepos, pkgRepo)
}
