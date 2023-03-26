package types

// PackageRepositories represents the repositories associated with a package
type PackageRepositories struct {
	Package
	Repositories []Repository `json:"repositories"`
}

// AddRepositories adds repositories. It will ignore any repositories that are
// already associated with the package.
func (pkg *PackageRepositories) AddRepositories(repos ...Repository) {
	for _, repo := range repos {
		if containsRepo(pkg.Repositories, repo) {
			continue
		}

		pkg.Repositories = append(pkg.Repositories, repo)
	}
}

func containsRepo(repos []Repository, repo Repository) bool {
	for _, r := range repos {
		if r.Name == repo.Name {
			return true
		}
	}

	return false
}
