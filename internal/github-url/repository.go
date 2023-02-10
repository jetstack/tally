package github_url

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	ghRegex       = regexp.MustCompile(`(?:https|git)(?:://|@)github\.com[/:]([^/:#]+)/([^/#]*).*`)
	ghSuffixRegex = regexp.MustCompile(`(\.git/?)?(\.git|\?.*|#.*)?$`)
)

// ToRepository parses a github url from a number of different formats into our
// expected repository format: github.com/<org>/<repo>.
func ToRepository(u string) (string, error) {
	matches := ghRegex.FindStringSubmatch(ghSuffixRegex.ReplaceAllString(u, ""))
	if len(matches) < 3 {
		return "", fmt.Errorf("couldn't parse url")
	}
	return strings.Join([]string{"github.com", matches[1], matches[2]}, "/"), nil
}
