package bom

import (
	"errors"
	"strings"

	"github.com/jetstack/tally/internal/types"
	"github.com/package-url/packageurl-go"
)

var (
	// ErrUnsupportedPackageType is returned when parsing a package of an
	// unsupported type
	ErrUnsupportedPackageType = errors.New("unsupported package type")
)

func packageFromPurl(purl packageurl.PackageURL) (*types.Package, error) {
	switch purl.Type {
	case packageurl.TypeNPM:
		name := purl.Name
		if purl.Namespace != "" {
			name = purl.Namespace + "/" + name
		}
		return &types.Package{
			System: "NPM",
			Name:   name,
		}, nil
	case packageurl.TypeGolang:
		name := purl.Name
		if purl.Namespace != "" {
			name = purl.Namespace + "/" + purl.Name
		}
		return &types.Package{
			System: "GO",
			Name:   name,
		}, nil
	case packageurl.TypeMaven:
		return &types.Package{
			System: "MAVEN",
			Name:   strings.Join([]string{purl.Namespace, purl.Name}, ":"),
		}, nil
	case packageurl.TypePyPi:
		return &types.Package{
			System: "PYPI",
			Name:   purl.Name,
		}, nil
	case "cargo":
		return &types.Package{
			System: "CARGO",
			Name:   purl.Name,
		}, nil
	default:
		return nil, ErrUnsupportedPackageType
	}
}
