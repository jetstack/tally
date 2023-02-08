package repositories

import (
	"context"
	"strings"

	"github.com/jetstack/tally/internal/types"
)

// PackageMapper is a mapper that infers repositories from information that is
// apparent from the package, i.e the name
var PackageMapper Mapper = &pkgMapper{}

type pkgMapper struct{}

// Repositories tries to infer repositories from information that is apparent
// from the package itself
func (m *pkgMapper) Repositories(ctx context.Context, pkg types.Package) ([]string, error) {
	switch pkg.System {
	case "GO":
		if !strings.HasPrefix(pkg.Name, "github.com/") {
			return []string{}, nil
		}
		parts := strings.Split(pkg.Name, "/")
		if len(parts) < 3 {
			return []string{}, nil
		}

		return []string{strings.Join([]string{parts[0], parts[1], parts[2]}, "/")}, nil
	default:
		return []string{}, nil
	}
}
