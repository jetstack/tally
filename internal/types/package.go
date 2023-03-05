package types

// Package is a package
type Package struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// Equals compares one package to another
func (pkg *Package) Equals(p Package) bool {
	return pkg.Type == p.Type && pkg.Name == p.Name
}
