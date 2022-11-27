package types

// Packages is a list of packages
type Packages []*Package

// Add a package to the list
func (pkgs *Packages) Add(pkg *Package) {
	for _, p := range *pkgs {
		if p.Equals(pkg) {
			p.AddRepositories(pkg.Repositories...)

			return
		}
	}

	*pkgs = append(*pkgs, pkg)
}

// Package is a package
type Package struct {
	System       string   `json:"system"`
	Name         string   `json:"name"`
	Repositories []string `json:"repositories,omitempty"`
}

// Equals compares one package to another
func (pkg *Package) Equals(p *Package) bool {
	if pkg == nil && p == nil {
		return true
	}
	if p == nil {
		return false
	}

	return pkg.System == p.System && pkg.Name == p.Name
}

// AddRepositories adds repositories to the package
func (pkg *Package) AddRepositories(repos ...string) {
	for _, repo := range repos {
		if contains(pkg.Repositories, repo) {
			continue
		}

		pkg.Repositories = append(pkg.Repositories, repo)
	}
}

func contains(values []string, val string) bool {
	for _, v := range values {
		if v == val {
			return true
		}
	}

	return false
}
