package bom

import (
	"encoding/json"
	"io"

	"github.com/anchore/syft/syft/formats/syftjson/model"
	syft "github.com/anchore/syft/syft/pkg"
	github_url "github.com/jetstack/tally/internal/github-url"
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

// PackagesFromSyftBOM discovers packages in a Syft BOM
func PackagesFromSyftBOM(doc *model.Document) ([]Package, error) {
	var pkgs []Package
	for _, a := range doc.Artifacts {
		if a.PURL == "" {
			continue
		}

		pkg, err := packageFromPurl(a.PURL)
		if err != nil {
			return nil, err
		}
		if pkg == nil {
			continue
		}

		repos := repositoriesFromSyftPackage(a)
		for _, repo := range repos {
			if contains(pkg.Repositories, repo) {
				continue
			}
			pkg.Repositories = append(pkg.Repositories, repo)
		}

		pkgs = appendPackage(pkgs, *pkg)
	}

	return pkgs, nil
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

func appendPackage(pkgs []Package, pkg Package) []Package {
	for i, p := range pkgs {
		if p.Type != pkg.Type {
			continue
		}
		if p.Name != pkg.Name {
			continue
		}

		for _, repo := range pkg.Repositories {
			if contains(p.Repositories, repo) {
				continue
			}

			pkgs[i].Repositories = append(pkgs[i].Repositories, repo)
		}

		return pkgs
	}

	return append(pkgs, pkg)
}
