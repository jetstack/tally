package repositories

import (
	"context"

	"github.com/jetstack/tally/internal/bom"
	"github.com/jetstack/tally/internal/types"
)

type bomMapper struct {
	bom bom.BOM
}

// BOMMapper returns a mapper that extracts repositories from a BOM
func BOMMapper(bom bom.BOM) Mapper {
	return &bomMapper{
		bom: bom,
	}
}

// Repositories returns any repositories for components in a BOM
func (m *bomMapper) Repositories(ctx context.Context, pkg types.Package) ([]string, error) {
	return m.bom.Repositories(pkg)
}
