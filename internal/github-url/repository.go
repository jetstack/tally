package github_url

import (
	"regexp"
	"strings"

	"github.com/jetstack/tally/internal/types"
)

var (
	ghRegex       = regexp.MustCompile(`(?:(?:https|git)(?:://|@))?github\.com[/:]([^/:#]+)/([^/#]*).*`)
	ghSuffixRegex = regexp.MustCompile(`(\.git/?)?(\.git|\?.*|#.*)?$`)
)

// ToRepository parses a github url from a number of different formats into our
// expected repository format: github.com/<org>/<repo>.
func ToRepository(u string) *types.Repository {
	matches := ghRegex.FindStringSubmatch(ghSuffixRegex.ReplaceAllString(u, ""))
	if len(matches) < 3 {
		return nil
	}

	return &types.Repository{
		Name: strings.Join([]string{"github.com", matches[1], matches[2]}, "/"),
	}
}
