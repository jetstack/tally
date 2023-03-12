package cache

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/scorecard"
	"github.com/ossf/scorecard-webapp/app/generated/models"
)

type mockCache struct {
	repoToScorecardResult map[string]*models.ScorecardResult
	putErr                error
	getErr                error
}

func (c *mockCache) GetResult(ctx context.Context, repository string) (*models.ScorecardResult, error) {
	if c.getErr != nil {
		return nil, c.getErr
	}

	score, ok := c.repoToScorecardResult[repository]
	if !ok {
		return nil, ErrNotFound
	}

	return score, nil
}

func (c *mockCache) PutResult(ctx context.Context, repository string, result *models.ScorecardResult) error {
	if c.putErr != nil {
		return c.putErr
	}

	c.repoToScorecardResult[repository] = result

	return nil
}

type mockScorecardClient struct {
	repoToScorecardResult map[string]*models.ScorecardResult
	getErr                error
	name                  string
}

func (c *mockScorecardClient) Name() string {
	return c.name
}

func (c *mockScorecardClient) GetResult(ctx context.Context, repository string) (*models.ScorecardResult, error) {
	if c.getErr != nil {
		return nil, c.getErr
	}

	result, ok := c.repoToScorecardResult[repository]
	if !ok {
		return nil, ErrNotFound
	}

	return result, nil
}

func TestScorecardClientGetScore(t *testing.T) {
	type testCase struct {
		repository          string
		cache               Cache
		scorecardClient     scorecard.Client
		wantScorecardResult *models.ScorecardResult
		wantErr             error
	}
	testCases := map[string]func(t *testing.T) *testCase{
		"should return score from cache": func(t *testing.T) *testCase {
			repository := "github.com/foo/bar"

			wantScorecardResult := &models.ScorecardResult{
				Repo: &models.Repo{
					Name: repository,
				},
				Score: 7.2,
				Checks: []*models.ScorecardCheck{
					{
						Name:  "foo",
						Score: 8,
					},
					{
						Name:  "bar",
						Score: 2,
					},
				},
			}
			return &testCase{
				repository: repository,
				cache: &mockCache{
					repoToScorecardResult: map[string]*models.ScorecardResult{
						repository: wantScorecardResult,
					},
				},
				scorecardClient: &mockScorecardClient{
					repoToScorecardResult: map[string]*models.ScorecardResult{
						repository: {
							Score: 2.1,
						},
					},
				},
				wantScorecardResult: wantScorecardResult,
			}
		},
		"should return score from client when cache returns ErrNotFound": func(t *testing.T) *testCase {
			repository := "github.com/foo/bar"
			wantScorecardResult := &models.ScorecardResult{
				Repo: &models.Repo{
					Name: repository,
				},
				Score: 7.2,
				Checks: []*models.ScorecardCheck{
					{
						Name:  "foo",
						Score: 8,
					},
					{
						Name:  "bar",
						Score: 2,
					},
				},
			}
			return &testCase{
				repository: repository,
				cache: &mockCache{
					repoToScorecardResult: map[string]*models.ScorecardResult{},
				},
				scorecardClient: &mockScorecardClient{
					repoToScorecardResult: map[string]*models.ScorecardResult{
						repository: wantScorecardResult,
					},
				},
				wantScorecardResult: wantScorecardResult,
			}
		},
		"should return error from cache": func(t *testing.T) *testCase {
			repository := "github.com/foo/bar"
			wantErr := errors.New("foobar")
			return &testCase{
				repository: repository,
				cache: &mockCache{
					getErr: wantErr,
				},
				scorecardClient: &mockScorecardClient{},
				wantErr:         wantErr,
			}
		},
		"should return error from client": func(t *testing.T) *testCase {
			repository := "github.com/foo/bar"
			wantErr := errors.New("foobar")
			return &testCase{
				repository: repository,
				cache:      &mockCache{},
				scorecardClient: &mockScorecardClient{
					getErr: wantErr,
				},
				wantErr: wantErr,
			}
		},
		"shouldn't return error when cache has score": func(t *testing.T) *testCase {
			repository := "github.com/foo/bar"
			wantScorecardResult := &models.ScorecardResult{
				Repo: &models.Repo{
					Name: repository,
				},
				Score: 7.2,
				Checks: []*models.ScorecardCheck{
					{
						Name:  "foo",
						Score: 8,
					},
					{
						Name:  "bar",
						Score: 2,
					},
				},
			}
			return &testCase{
				repository: repository,
				cache: &mockCache{
					repoToScorecardResult: map[string]*models.ScorecardResult{
						repository: wantScorecardResult,
					},
				},
				scorecardClient: &mockScorecardClient{
					getErr: errors.New("foobar"),
				},
				wantScorecardResult: wantScorecardResult,
			}
		},
		"should return ErrNotFound when score not found in cache or client": func(t *testing.T) *testCase {
			repository := "github.com/foo/bar"
			return &testCase{
				repository:      repository,
				cache:           &mockCache{},
				scorecardClient: &mockScorecardClient{},
				wantErr:         ErrNotFound,
			}
		},
	}
	for n, setup := range testCases {
		t.Run(n, func(t *testing.T) {
			tc := setup(t)

			gotScorecardResult, err := NewScorecardClient(tc.cache, tc.scorecardClient).GetResult(context.Background(), tc.repository)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(tc.wantScorecardResult, gotScorecardResult); diff != "" {
				t.Errorf("unexpected score:\n%s", diff)
			}
		})
	}
}

func TestScorecardClientName(t *testing.T) {
	wantName := "foobar"
	gotName := NewScorecardClient(&mockCache{}, &mockScorecardClient{name: wantName}).Name()
	if gotName != wantName {
		t.Errorf("unexpected name returned by caching client; wanted %s but got %s", wantName, gotName)
	}
}
