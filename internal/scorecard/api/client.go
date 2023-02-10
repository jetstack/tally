package scorecard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jetstack/tally/internal/scorecard"
	"github.com/jetstack/tally/internal/types"
	"github.com/ossf/scorecard-webapp/app/generated/models"
)

const (
	// DefaultURL is the default base url for the scorecard API
	DefaultURL = "https://api.securityscorecards.dev"

	// DefaultTimeout is the default timeout for HTTP requests
	DefaultTimeout = time.Second * 30
)

// Client retrieves scores from the scorecard API
type Client struct {
	baseURL       *url.URL
	httpClient    *http.Client
	githubBaseURL string
}

// NewClient returns a client that fetches scores from the scorecard API
func NewClient(rawURL string, opts ...Option) (scorecard.Client, error) {
	if rawURL == "" {
		rawURL = DefaultURL
	}
	baseURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parsing url: %w", err)
	}
	c := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// GetScore fetches a score from the public scorecard API
func (c *Client) GetScore(ctx context.Context, repository string) (*types.Score, error) {
	parts := strings.Split(repository, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("unexpected number of parts in %s; wanted 3 but got %d: %w", repository, len(parts), scorecard.ErrInvalidRepository)
	}

	switch parts[0] {
	// Ensure the repository is public before checking for a score. We want to
	// avoid exposing private repositories to the API, as some users may
	// consider that sensitive information they may not want shipping off to
	// the Scorecard project.
	case "github.com":
		baseURL := c.githubBaseURL
		if baseURL == "" {
			baseURL = fmt.Sprintf("https://%s", parts[0])
		}
		resp, err := c.httpClient.Head(fmt.Sprintf("%s/%s/%s", baseURL, parts[1], parts[2]))
		if err != nil {
			return nil, fmt.Errorf("error checking if repository is public: %w", errors.Join(scorecard.ErrUnexpectedResponse, err))
		}
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("github repository not found: %w", scorecard.ErrNotFound)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("non-200 response from github when checking repository: %w", scorecard.ErrUnexpectedResponse)
		}
	default:
		return nil, fmt.Errorf("unsupported repository platform %s: %w", parts[0], scorecard.ErrInvalidRepository)
	}

	// Get result from the Scorecard API
	result, err := c.getResult(parts[0], parts[1], parts[2])
	if err != nil {
		return nil, fmt.Errorf("fetching result: %w", err)
	}

	score := &types.Score{
		Score:  result.Score,
		Checks: map[string]int{},
	}
	for _, check := range result.Checks {
		if check == nil {
			continue
		}
		score.Checks[check.Name] = int(check.Score)
	}

	return score, nil
}

// ConcurrencyLimit indicates that the client can be ran concurrently
func (c *Client) ConcurrencyLimit() int {
	return 0
}

func (c *Client) getResult(platform, org, repo string) (*models.ScorecardResult, error) {
	uri, err := c.baseURL.Parse(fmt.Sprintf("/projects/%s/%s/%s", platform, org, repo))
	if err != nil {
		return nil, fmt.Errorf("parsing path: %w", err)
	}
	resp, err := c.httpClient.Get(uri.String())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", uri, errors.Join(err, scorecard.ErrUnexpectedResponse))
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%s: %d: %w", uri, resp.StatusCode, scorecard.ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %d: %w", uri, resp.StatusCode, scorecard.ErrUnexpectedResponse)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body from %s: %w", uri, err)
	}
	result := &models.ScorecardResult{}
	if err := json.Unmarshal(body, result); err != nil {
		return nil, fmt.Errorf("unmarshaling json from %s: %w", uri, err)
	}

	return result, nil
}
