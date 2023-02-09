package cache

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/scorecard"
	"github.com/jetstack/tally/internal/types"
)

type mockCache struct {
	repoToScore map[string]*types.Score
	putErr      error
	getErr      error
}

func (c *mockCache) GetScore(ctx context.Context, repository string) (*types.Score, error) {
	if c.getErr != nil {
		return nil, c.getErr
	}

	score, ok := c.repoToScore[repository]
	if !ok {
		return nil, ErrNotFound
	}

	return score, nil
}

func (c *mockCache) PutScore(ctx context.Context, repository string, score *types.Score) error {
	if c.putErr != nil {
		return c.putErr
	}

	c.repoToScore[repository] = score

	return nil
}

type mockScorecardClient struct {
	repoToScore map[string]*types.Score
	getErr      error
	name        string
	limit       int
}

func (c *mockScorecardClient) Name() string {
	return c.name
}

func (c *mockScorecardClient) ConcurrencyLimit() int {
	return c.limit
}

func (c *mockScorecardClient) GetScore(ctx context.Context, repository string) (*types.Score, error) {
	if c.getErr != nil {
		return nil, c.getErr
	}

	score, ok := c.repoToScore[repository]
	if !ok {
		return nil, ErrNotFound
	}

	return score, nil
}

func TestScorecardClientGetScore(t *testing.T) {
	type testCase struct {
		repository      string
		cache           Cache
		scorecardClient scorecard.Client
		wantScore       *types.Score
		wantErr         error
	}
	testCases := map[string]func(t *testing.T) *testCase{
		"should return score from cache": func(t *testing.T) *testCase {
			repository := "github.com/foo/bar"
			wantScore := &types.Score{
				Score: 4.5,
				Checks: map[string]int{
					"foo": 3,
					"bar": 7,
				},
			}
			return &testCase{
				repository: repository,
				cache: &mockCache{
					repoToScore: map[string]*types.Score{
						repository: wantScore,
					},
				},
				scorecardClient: &mockScorecardClient{
					repoToScore: map[string]*types.Score{
						repository: {
							Score: 2.1,
						},
					},
				},
				wantScore: wantScore,
			}
		},
		"should return score from client when cache returns ErrNotFound": func(t *testing.T) *testCase {
			repository := "github.com/foo/bar"
			wantScore := &types.Score{
				Score: 4.5,
				Checks: map[string]int{
					"foo": 3,
					"bar": 7,
				},
			}
			return &testCase{
				repository: repository,
				cache: &mockCache{
					repoToScore: map[string]*types.Score{},
				},
				scorecardClient: &mockScorecardClient{
					repoToScore: map[string]*types.Score{
						repository: wantScore,
					},
				},
				wantScore: wantScore,
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
			wantScore := &types.Score{
				Score: 4.5,
				Checks: map[string]int{
					"foo": 3,
					"bar": 7,
				},
			}
			return &testCase{
				repository: repository,
				cache: &mockCache{
					repoToScore: map[string]*types.Score{
						repository: wantScore,
					},
				},
				scorecardClient: &mockScorecardClient{
					getErr: errors.New("foobar"),
				},
				wantScore: wantScore,
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

			gotScore, err := NewScorecardClient(tc.cache, tc.scorecardClient).GetScore(context.Background(), tc.repository)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(tc.wantScore, gotScore); diff != "" {
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

func TestScorecardClientConcurrencyLimit(t *testing.T) {
	wantLimit := 13
	gotLimit := NewScorecardClient(&mockCache{}, &mockScorecardClient{limit: wantLimit}).ConcurrencyLimit()
	if gotLimit != wantLimit {
		t.Errorf("unexpected concurrency limit returned by caching client; wanted %d but got %d", wantLimit, gotLimit)
	}
}
