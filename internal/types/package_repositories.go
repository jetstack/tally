package types

type PackageRepositories struct {
	Package
	Repositories []string `json:"repositories"`
}

func (pkg *PackageRepositories) AddRepositories(repos ...string) {
	for _, repo := range repos {
		if contains(pkg.Repositories, repo) {
			continue
		}

		pkg.Repositories = append(pkg.Repositories, repo)
	}
}

func contains(vals []string, val string) bool {
	for _, v := range vals {
		if v == val {
			return true
		}
	}

	return false
}
