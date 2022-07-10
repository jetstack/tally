package tally

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"github.com/ossf/scorecard/v4/checker"
	"github.com/ossf/scorecard/v4/checks"
	docs "github.com/ossf/scorecard/v4/docs/checks"
	"github.com/ossf/scorecard/v4/pkg"
	"google.golang.org/api/iterator"
)

// Package is a package with score information
type Package struct {
	System         string     `json:"system"`
	Name           string     `json:"name"`
	Version        string     `json:"version"`
	RepositoryName string     `json:"repository,omitempty"`
	Score          float64    `json:"score,omitempty"`
	Date           civil.Date `json:"date,omitempty"`
}

// MarshalJSON implements json.Marshaler. It marshals the date field into an
// empty string rather than the default zero value of civil.Date.
func (p *Package) MarshalJSON() ([]byte, error) {
	alias := struct {
		System         string  `json:"system"`
		Name           string  `json:"name"`
		Version        string  `json:"version"`
		RepositoryName string  `json:"repository,omitempty"`
		Score          float64 `json:"score,omitempty"`
		Date           string  `json:"date,omitempty"`
	}{
		System:         p.System,
		Name:           p.Name,
		Version:        p.Version,
		RepositoryName: p.RepositoryName,
		Score:          p.Score,
	}
	if !p.Date.IsZero() {
		alias.Date = p.Date.String()
	}

	return json.Marshal(&alias)
}

// AddRepositoriesFromDepsDev encriches the provided packages with repository
// information taken from the deps.dev dataset.
func AddRepositoriesFromDepsDev(ctx context.Context, bq *bigquery.Client, pkgs []Package) ([]Package, error) {
	var pkgQ []string
	for _, pkg := range pkgs {
		if pkg.RepositoryName != "" {
			continue
		}
		pkgQ = append(pkgQ, fmt.Sprintf("('%s', '%s', '%s')", pkg.System, pkg.Name, pkg.Version))
	}

	q := bq.Query(fmt.Sprintf(`
SELECT DISTINCT concat('github.com/', projectname) as repositoryname, system, name, version
FROM `+"`bigquery-public-data.deps_dev_v1.PackageVersionToProject`"+` 
WHERE projecttype = 'GITHUB' 
AND (system,name,version) in (%s);
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

	// Include the packages that don't have a repository
	for i, pkg := range pkgs {
		for _, sPkg := range sPkgs {
			if isPackage(pkg, sPkg) {
				pkgs[i].RepositoryName = sPkg.RepositoryName
			}
		}
	}

	return pkgs, nil
}

// AddScoresFromScorecardLatest encriches the provided packages with scores from
// the latest scorecard dataset.
func AddScoresFromScorecardLatest(ctx context.Context, bq *bigquery.Client, pkgs []Package) ([]Package, error) {
	var repositories []string
	for _, pkg := range pkgs {
		if pkg.RepositoryName == "" {
			continue
		}
		repositories = append(repositories, pkg.RepositoryName)
	}

	q := bq.Query(fmt.Sprintf(`
SELECT repo.name as repositoryname, score, date
FROM `+"`openssf.scorecardcron.scorecard-v2_latest`"+`
WHERE repo.name IN ('%s');
`,
		strings.Join(repositories, "', '"),
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

	// Add scores to the list of packages
	for _, sPkg := range sPkgs {
		for i, pkg := range pkgs {
			if pkg.RepositoryName == sPkg.RepositoryName {
				pkgs[i].Score = sPkg.Score
				pkgs[i].Date = sPkg.Date
			}
		}
	}

	return pkgs, nil
}

// GenerateScoresForPackages generates scores for the packages that have a
// repository value but no score.
func GenerateScoresForPackages(ctx context.Context, pkgs []Package) ([]Package, error) {
	repoScores := map[string]float64{}
	for i, pkg := range pkgs {
		if !(strings.HasPrefix(pkg.RepositoryName, "github.com/") && pkg.Score == 0) {
			continue
		}

		var (
			score float64
			err   error
			ok    bool
		)
		score, ok = repoScores[pkg.RepositoryName]
		if !ok {
			// Genererate a score and add it to the package
			score, err = generateScoreForRepository(ctx, pkg.RepositoryName)
			if err != nil {
				return nil, err
			}
		}

		pkgs[i].Score = score
		pkgs[i].Date = civil.DateOf(time.Now())
	}

	return pkgs, nil
}

func generateScoreForRepository(ctx context.Context, repository string) (float64, error) {
	repoURI, repoClient, ossFuzzRepoClient, ciiClient, vulnsClient, err := checker.GetClients(
		ctx, repository, "", nil)
	if err != nil {
		return 0, err
	}
	defer repoClient.Close()
	if ossFuzzRepoClient != nil {
		defer ossFuzzRepoClient.Close()
	}

	enabledChecks := checks.GetAll()

	checkDocs, err := docs.Read()
	if err != nil {
		return 0, err
	}

	res, err := pkg.RunScorecards(
		ctx,
		repoURI,
		"HEAD",
		enabledChecks,
		repoClient,
		ossFuzzRepoClient,
		ciiClient,
		vulnsClient,
	)
	if err != nil {
		return 0, err
	}

	return res.GetAggregateScore(checkDocs)
}

func isPackage(a, b Package) bool {
	return a.System == b.System && a.Name == b.Name && a.Version == b.Version
}

func containsPackage(pkgs []Package, pkg Package) bool {
	for _, p := range pkgs {
		if isPackage(p, pkg) {
			return true
		}
	}

	return false
}
