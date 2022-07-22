package depsdev

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

// PackageVersion is a version of a package in the deps.dev dataset
type PackageVersion struct {
	System  string
	Name    string
	Version string
}

// PackageVersionToProject is the association between a package version and a
// project
type PackageVersionToProject struct {
	System      string
	Name        string
	Version     string
	ProjectName string
}

// PackageVersionToProjectTable is the table in the deps.dev dataset that we use
// to associate a package with a project
type PackageVersionToProjectTable interface {
	SelectGithubProjectsWherePackageVersions(context.Context, []PackageVersion) ([]PackageVersionToProject, error)
	String() string
}

type packageVersionToProjectTable struct {
	bq *bigquery.Client
}

// NewPackageVersionToProjectTable returns a table from the given reference
func NewPackageVersionToProjectTable(bq *bigquery.Client) PackageVersionToProjectTable {
	return &packageVersionToProjectTable{bq}
}

// String returns a string representation of the table in the form:
// <project-id>.<dataset-name>.<table-name>
func (t *packageVersionToProjectTable) String() string {
	return "bigquery-public-data.deps_dev_v1.PackageVersionToProject"
}

// SelectGithubProjectsWherePackageVersions selects github projects for the
// provided package versions
func (t *packageVersionToProjectTable) SelectGithubProjectsWherePackageVersions(ctx context.Context, pkgs []PackageVersion) ([]PackageVersionToProject, error) {
	var pkgQ []string
	for _, pkg := range pkgs {
		pkgQ = append(pkgQ, fmt.Sprintf("('%s', '%s', '%s')", pkg.System, pkg.Name, pkg.Version))
	}

	q := t.bq.Query(fmt.Sprintf(`
SELECT DISTINCT projectname, system, name, version
FROM `+fmt.Sprintf("`%s`", t)+` 
WHERE projecttype = 'GITHUB' 
AND (system,name,version) in (%s);
`,
		strings.Join(pkgQ, ","),
	))

	it, err := q.Read(ctx)
	if err != nil {
		return nil, err
	}

	var rows []PackageVersionToProject
	for {
		var row PackageVersionToProject
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		rows = append(rows, row)
	}

	return rows, nil
}
