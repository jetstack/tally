package bom

import (
	"strings"

	"github.com/package-url/packageurl-go"
)

func packageFromPurl(purl string) (*Package, error) {
	p, err := packageurl.FromString(purl)
	if err != nil {
		return nil, err
	}
	pkg := &Package{
		Type: p.Type,
		Name: p.Name,
	}
	if p.Namespace != "" {
		pkg.Name = p.Namespace + "/" + p.Name
	}

	switch pkg.Type {
	case "golang":
		if !strings.HasPrefix(pkg.Name, "github.com/") {
			return pkg, nil
		}
		parts := strings.Split(pkg.Name, "/")
		if len(parts) < 3 {
			return pkg, nil
		}

		pkg.Repositories = append(pkg.Repositories, strings.Join([]string{parts[0], parts[1], parts[2]}, "/"))
	}

	return pkg, nil
}
