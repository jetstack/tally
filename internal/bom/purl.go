package bom

import (
	"strings"

	"github.com/jetstack/tally/internal/types"
	"github.com/package-url/packageurl-go"
)

func packageRepositoriesFromPurl(purl string) (*types.PackageRepositories, error) {
	p, err := packageurl.FromString(purl)
	if err != nil {
		return nil, err
	}
	pkgRepo := &types.PackageRepositories{
		Package: types.Package{
			Type: p.Type,
			Name: p.Name,
		},
	}
	if p.Namespace != "" {
		pkgRepo.Name = p.Namespace + "/" + p.Name
	}

	switch pkgRepo.Type {
	case "golang":
		if !strings.HasPrefix(pkgRepo.Name, "github.com/") {
			return pkgRepo, nil
		}
		parts := strings.Split(pkgRepo.Name, "/")
		if len(parts) < 3 {
			return pkgRepo, nil
		}

		pkgRepo.AddRepositories(strings.Join([]string{parts[0], parts[1], parts[2]}, "/"))
	}

	return pkgRepo, nil
}
