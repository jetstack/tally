package github_url

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/types"
)

func TestToRepository(t *testing.T) {
	testCases := []struct {
		url      string
		wantRepo *types.Repository
	}{
		{
			url: "github.com/foo/bar",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "https://github.com/foo/bar",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "https://github.com/foo/bar/tree/main/baz",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "https://github.com/foo/bar#baz",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "https://github.com/foo/bar/",
			wantRepo: &types.Repository{
				"github.com/foo/bar",
			},
		},
		{
			url: "https://github.com/foo/bar.git",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "https://github.com/foo/bar.git/",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "https://github.com/foo/bar.git?ref=baz",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "https://github.com/foo/bar.git?ref=baz",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "https://github.com/foo/bar?ref=something",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "https://github.com/foo/bar#something",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "git://github.com/foo/bar",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "git://github.com/foo/bar",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "git://github.com/foo/bar/",
			wantRepo: &types.Repository{
				"github.com/foo/bar",
			},
		},
		{
			url: "git://github.com/foo/bar.git",
			wantRepo: &types.Repository{
				"github.com/foo/bar",
			},
		},
		{
			url: "git://github.com/foo/bar.git/",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "git://github.com/foo/bar.git?ref=baz",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "git://github.com/foo/bar?ref=something",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "git://github.com/foo/bar#something",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "git@github.com:foo/bar.git",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "git@github.com:foo/bar.git",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "git@github.com:foo/bar.git/",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "git@github.com:foo/bar.git?ref=something",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "git@github.com:foo/bar.git#something",
			wantRepo: &types.Repository{
				Name: "github.com/foo/bar",
			},
		},
		{
			url: "https://github.com/foo",
		},
		{
			url: "https://gitlab.com/foo/bar",
		},

		{
			url: "git://gitlab.com/foo/bar.git",
		},
		{
			url: "git@gitlab.com:foo/bar.git",
		},
		{
			url: "github.com",
		},
	}
	for _, tc := range testCases {
		gotRepo := ToRepository(tc.url)

		if diff := cmp.Diff(gotRepo, tc.wantRepo); diff != "" {
			t.Errorf("unexpected repository:\n%s", diff)
		}
	}
}
