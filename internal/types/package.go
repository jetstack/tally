package types

// Package is a package
type Package struct {
	System string `json:"system"`
	Name   string `json:"name"`
}

// Equals compares one package to another
func (pkg *Package) Equals(p Package) bool {
	return pkg.System == p.System && pkg.Name == p.Name
}
