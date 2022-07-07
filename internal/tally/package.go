package tally

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"google.golang.org/api/iterator"
)

// Package is a package with score information
type Package struct {
	System         string
	Name           string
	Version        string
	RepositoryName string
	Score          float64
	Date           civil.Date
}

// ScorePackages enriches the provided list of packages with scores taken from
// the OpenSSF scorecard data set. The returned packages are sorted by score in
// descending order.
func ScorePackages(ctx context.Context, bq *bigquery.Client, pkgs []Package) ([]Package, error) {
	var pkgQ []string
	for _, pkg := range pkgs {
		pkgQ = append(pkgQ, fmt.Sprintf("('%s', '%s', '%s')", pkg.System, pkg.Name, pkg.Version))
	}

	// TODO: find a SQL wizard to explain to me how I could make this query
	// more efficient
	q := bq.Query(fmt.Sprintf(`
SELECT system, name, version, repo.name as repositoryname, score, date
FROM `+"`openssf.scorecardcron.scorecard-v2`"+` AS t1
INNER JOIN (
  SELECT DISTINCT concat('github.com/', ProjectName) as reponame, system, name, version
  FROM `+"`bigquery-public-data.deps_dev_v1.PackageVersionToProject`"+` 
  WHERE projecttype = 'GITHUB' 
  AND (system,name,version) in (%s)
) t2
ON t1.repo.name = t2.reponame
WHERE NOT EXISTS (
  SELECT *
  FROM `+"`openssf.scorecardcron.scorecard-v2`"+` AS t3
  WHERE t3.repo.name = t1.repo.name AND t3.date > t1.date
);
`,
		strings.Join(pkgQ, ","),
	))

	it, err := q.Read(ctx)
	if err != nil {
		return nil, err
	}

	var sPkgs []Package
	for {
		var sPkg Package
		err := it.Next(&sPkg)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		sPkgs = append(sPkgs, sPkg)
	}

	// Include the packages that don't have a score
	for _, pkg := range pkgs {
		if containsPackage(sPkgs, pkg) {
			continue
		}

		sPkgs = append(sPkgs, pkg)
	}

	// Sort the packages by score
	sort.Slice(sPkgs, func(i, j int) bool {
		return sPkgs[i].Score > sPkgs[j].Score
	})

	return sPkgs, nil
}

func containsPackage(pkgs []Package, pkg Package) bool {
	for _, p := range pkgs {
		if p.System == pkg.System && p.Name == pkg.Name && p.Version == pkg.Version {
			return true
		}
	}

	return false
}
