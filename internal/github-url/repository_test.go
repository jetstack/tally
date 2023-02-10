package github_url

import "testing"

func TestToRepository(t *testing.T) {
	testCases := []struct {
		url      string
		wantRepo string
		wantErr  bool
	}{
		{
			url:      "https://github.com/foo/bar",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "https://github.com/foo/bar/tree/main/baz",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "https://github.com/foo/bar#baz",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "https://github.com/foo/bar/",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "https://github.com/foo/bar.git",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "https://github.com/foo/bar.git/",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "https://github.com/foo/bar.git?ref=baz",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "https://github.com/foo/bar.git?ref=baz",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "https://github.com/foo/bar?ref=something",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "https://github.com/foo/bar#something",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git://github.com/foo/bar",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git://github.com/foo/bar",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git://github.com/foo/bar/",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git://github.com/foo/bar.git",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git://github.com/foo/bar.git/",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git://github.com/foo/bar.git?ref=baz",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git://github.com/foo/bar?ref=something",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git://github.com/foo/bar#something",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git@github.com:foo/bar.git",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git@github.com:foo/bar.git",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git@github.com:foo/bar.git/",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git@github.com:foo/bar.git?ref=something",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:      "git@github.com:foo/bar.git#something",
			wantRepo: "github.com/foo/bar",
		},
		{
			url:     "https://github.com/foo",
			wantErr: true,
		},
		{
			url:     "https://gitlab.com/foo/bar",
			wantErr: true,
		},

		{
			url:     "git://gitlab.com/foo/bar.git",
			wantErr: true,
		},
		{
			url:     "git@gitlab.com:foo/bar.git",
			wantErr: true,
		},
		{
			url:     "github.com",
			wantErr: true,
		},
	}
	for _, tc := range testCases {
		gotRepo, err := ToRepository(tc.url)
		if err != nil && !tc.wantErr {
			t.Fatalf("unexpected error parsing %q: %s", tc.url, err)
		}
		if err == nil && tc.wantErr {
			t.Fatalf("expected error but got nil")
		}

		if gotRepo != tc.wantRepo {
			t.Fatalf("unexpected repo parsing %q; got %q but wanted %q", tc.url, gotRepo, tc.wantRepo)
		}
	}
}
