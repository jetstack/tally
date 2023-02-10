package scorecard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/scorecard"
	"github.com/jetstack/tally/internal/types"
	"github.com/ossf/scorecard-webapp/app/generated/models"
)

func TestClientGetScore(t *testing.T) {
	type testCase struct {
		scorecardHandler func(w http.ResponseWriter, r *http.Request)
		githubHandler    func(w http.ResponseWriter, r *http.Request)
		repository       string
		wantErr          error
		wantScore        *types.Score
	}
	testCases := map[string]func(t *testing.T) *testCase{
		"should return expected score": func(t *testing.T) *testCase {
			platform := "github.com"
			org := "foo"
			repo := "bar"
			return &testCase{
				repository: strings.Join([]string{platform, org, repo}, "/"),
				scorecardHandler: func(w http.ResponseWriter, r *http.Request) {
					parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/projects/"), "/")
					if len(parts) != 3 {
						t.Fatalf("unexpected number of parts in request path; wanted 3 but got %d", len(parts))
					}
					if parts[0] != "github.com" {
						t.Fatalf("unexpected platform; wanted %s but got %s", platform, parts[0])
					}
					if parts[1] != org {
						t.Fatalf("unexpected platform; wanted %s but got %s", org, parts[1])
					}
					if parts[2] != repo {
						t.Fatalf("unexpected platform; wanted %s but got %s", repo, parts[2])
					}

					result := &models.ScorecardResult{
						Repo: &models.Repo{
							Name: fmt.Sprintf("%s/%s/%s", platform, org, repo),
						},
						Score: 6.5,
						Checks: []*models.ScorecardCheck{
							{
								Name:  "foo",
								Score: 7,
							},
							{
								Name:  "bar",
								Score: 5,
							},
						},
					}
					w.Header().Set("Content-type", "application/json")
					json.NewEncoder(w).Encode(result)
				},
				githubHandler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
				wantScore: &types.Score{
					Score: 6.5,
					Checks: map[string]int{
						"foo": 7,
						"bar": 5,
					},
				},
			}
		},
		"should return scorecard.ErrNotFound when API returns a 404 response": func(t *testing.T) *testCase {
			return &testCase{
				repository: "github.com/foo/bar",
				scorecardHandler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				},
				githubHandler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
				wantErr: scorecard.ErrNotFound,
			}
		},
		"should return scorecard.ErrUnexpectedResponse when API returns a 500 response": func(t *testing.T) *testCase {
			return &testCase{
				repository: "github.com/foo/bar",
				scorecardHandler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
				githubHandler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
				wantErr: scorecard.ErrUnexpectedResponse,
			}
		},
		"should return scorecard.ErrInvalidRepository when an invalid repository string is provided": func(t *testing.T) *testCase {
			return &testCase{
				repository: "github.com/foo/bar/main",
				scorecardHandler: func(w http.ResponseWriter, r *http.Request) {
					t.Fatalf("unexpected call to scorecard API")
				},
				githubHandler: func(w http.ResponseWriter, r *http.Request) {
					t.Fatalf("unexpected call to github API")
				},
				wantErr: scorecard.ErrInvalidRepository,
			}
		},
		"should return scorecard.ErrInvalidRepository for a non-github repo": func(t *testing.T) *testCase {
			return &testCase{
				repository: "gitlab.com/foo/bar",
				scorecardHandler: func(w http.ResponseWriter, r *http.Request) {
					t.Fatalf("unexpected call to scorecard API")
				},
				githubHandler: func(w http.ResponseWriter, r *http.Request) {
					t.Fatalf("unexpected call to github API")
				},
				wantErr: scorecard.ErrInvalidRepository,
			}
		},
		"should return scorecard.ErrNotFound when Github returns a 404": func(t *testing.T) *testCase {
			return &testCase{
				repository: "github.com/foo/bar",
				scorecardHandler: func(w http.ResponseWriter, r *http.Request) {
					t.Fatalf("unexpected call to scorecard API")
				},
				githubHandler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				},
				wantErr: scorecard.ErrNotFound,
			}
		},
		"should return scorecard.ErrUnexpectedResponse when Github returns a 500": func(t *testing.T) *testCase {
			return &testCase{
				repository: "github.com/foo/bar",
				scorecardHandler: func(w http.ResponseWriter, r *http.Request) {
					t.Fatalf("unexpected call to scorecard API")
				},
				githubHandler: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
				wantErr: scorecard.ErrUnexpectedResponse,
			}
		},
	}
	for n, setup := range testCases {
		t.Run(n, func(t *testing.T) {
			tc := setup(t)

			mux := http.NewServeMux()
			server := httptest.NewServer(mux)
			mux.HandleFunc("/projects/", tc.scorecardHandler)

			gMux := http.NewServeMux()
			gServer := httptest.NewServer(gMux)
			gMux.HandleFunc("/", tc.githubHandler)

			opt := func(c *Client) {
				c.githubBaseURL = gServer.URL
			}
			c, err := NewClient(server.URL, opt)
			if err != nil {
				t.Fatalf("unexpected error creating client: %s", err)
			}

			gotScore, err := c.GetScore(context.Background(), tc.repository)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(tc.wantScore, gotScore); diff != "" {
				t.Errorf("unexpected score:\n%s", diff)
			}
		})
	}
}
