package scorecard

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/db"
	"github.com/jetstack/tally/internal/scorecard"
	"github.com/jetstack/tally/internal/types"
)

type mockDBScoreReader struct {
	getChecks func(context.Context, string) ([]db.Check, error)
	getScores func(context.Context, ...string) ([]db.Score, error)
}

func (r *mockDBScoreReader) GetChecks(ctx context.Context, repository string) ([]db.Check, error) {
	return r.getChecks(ctx, repository)
}

func (r *mockDBScoreReader) GetScores(ctx context.Context, repositories ...string) ([]db.Score, error) {
	return r.getScores(ctx, repositories...)
}

func TestClientGetScore(t *testing.T) {
	type testCase struct {
		sr        db.ScoreReader
		wantScore *types.Score
		wantErr   error
	}
	testCases := map[string]func(t *testing.T) *testCase{
		"should return expected score": func(t *testing.T) *testCase {
			return &testCase{
				sr: &mockDBScoreReader{
					getScores: func(context.Context, ...string) ([]db.Score, error) {
						return []db.Score{
							{
								Score: 7.2,
							},
						}, nil
					},
					getChecks: func(context.Context, string) ([]db.Check, error) {
						return []db.Check{
							{
								Name:  "foo",
								Score: 8,
							},
							{
								Name:  "bar",
								Score: 2,
							},
						}, nil
					},
				},
				wantScore: &types.Score{
					Score: 7.2,
					Checks: map[string]int{
						"foo": 8,
						"bar": 2,
					},
				},
			}
		},
		"should return error when there is more than one score for a repository": func(t *testing.T) *testCase {
			return &testCase{
				sr: &mockDBScoreReader{
					getScores: func(context.Context, ...string) ([]db.Score, error) {
						return []db.Score{
							{
								Score: 7.2,
							},
							{
								Score: 3.9,
							},
						}, nil
					},
					getChecks: func(context.Context, string) ([]db.Check, error) {
						t.Fatalf("unexpected call to GetChecks")
						return nil, nil
					},
				},
				wantErr: scorecard.ErrUnexpectedResponse,
			}
		},
		"should returns errors from GetChecks": func(t *testing.T) *testCase {
			wantErr := errors.New("check error")
			return &testCase{
				sr: &mockDBScoreReader{
					getScores: func(context.Context, ...string) ([]db.Score, error) {
						return []db.Score{
							{
								Score: 7.2,
							},
						}, nil
					},
					getChecks: func(context.Context, string) ([]db.Check, error) {
						return nil, wantErr
					},
				},
				wantErr: wantErr,
			}
		},
		"should returns errors from GetScores": func(t *testing.T) *testCase {
			wantErr := errors.New("check error")
			return &testCase{
				sr: &mockDBScoreReader{
					getScores: func(context.Context, ...string) ([]db.Score, error) {
						return nil, wantErr
					},
					getChecks: func(context.Context, string) ([]db.Check, error) {
						t.Fatalf("unexpected call to GetChecks")
						return nil, nil
					},
				},
				wantErr: wantErr,
			}
		},
		"should return ErrNotFound when GetChecks returns db.ErrNotFound": func(t *testing.T) *testCase {
			return &testCase{
				sr: &mockDBScoreReader{
					getScores: func(context.Context, ...string) ([]db.Score, error) {
						return []db.Score{
							{
								Score: 7.2,
							},
						}, nil
					},
					getChecks: func(context.Context, string) ([]db.Check, error) {
						return nil, db.ErrNotFound
					},
				},
				wantErr: scorecard.ErrNotFound,
			}
		},
		"should return ErrNotFound when GetScores returns db.ErrNotFound": func(t *testing.T) *testCase {
			return &testCase{
				sr: &mockDBScoreReader{
					getScores: func(context.Context, ...string) ([]db.Score, error) {
						return nil, db.ErrNotFound
					},
					getChecks: func(context.Context, string) ([]db.Check, error) {
						t.Fatalf("unexpected call to GetChecks")
						return nil, nil
					},
				},
				wantErr: scorecard.ErrNotFound,
			}
		},
	}
	for n, setup := range testCases {
		t.Run(n, func(t *testing.T) {
			tc := setup(t)

			gotScore, err := NewClient(tc.sr).GetScore(context.Background(), "foobar")
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(tc.wantScore, gotScore); diff != "" {
				t.Errorf("unexpected score:\n%s", diff)
			}
		})
	}
}
