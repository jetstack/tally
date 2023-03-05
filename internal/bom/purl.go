package bom

import (
	"github.com/jetstack/tally/internal/types"
	"github.com/package-url/packageurl-go"
)

func packageFromPurl(purl string) (*types.Package, error) {
	p, err := packageurl.FromString(purl)
	if err != nil {
		return nil, err
	}
	pkg := &types.Package{
		Type: p.Type,
		Name: p.Name,
	}
	if p.Namespace != "" {
		pkg.Name = p.Namespace + "/" + p.Name
	}

	return pkg, nil
}
