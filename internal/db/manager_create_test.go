package db

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type mockSrc struct {
	name   string
	update func(context.Context, Writer) error
}

func (m *mockSrc) String() string {
	return m.name
}
func (m *mockSrc) Update(ctx context.Context, tallyDB Writer) error {
	if m.update == nil {
		return nil
	}

	return m.update(ctx, tallyDB)
}

func TestManagerCreateDB(t *testing.T) {
	dbDir, err := ioutil.TempDir("", "tally-db")
	if err != nil {
		t.Fatalf("unexpected error creating temp dir: %s", err)
	}
	defer os.RemoveAll(dbDir)

	mgr, err := NewManager(dbDir, ioutil.Discard)
	if err != nil {
		t.Fatalf("unexpected error creating manager: %s", err)
	}

	wantPkgs := map[Package][]string{
		{
			System: "GO",
			Name:   "foobar",
		}: {
			"github.com/baz/foo",
			"github.com/foo/bar",
		},
		{
			System: "NPM",
			Name:   "barfoo",
		}: {"github.com/bar/foo"},
	}

	src := &mockSrc{
		name: "mock",
		update: func(ctx context.Context, tallyDB Writer) error {
			var pkgs []Package
			for pkg, repos := range wantPkgs {
				for _, repo := range repos {
					pkgs = append(pkgs, Package{
						System:     pkg.System,
						Name:       pkg.Name,
						Repository: repo,
					})
				}
			}
			if err := tallyDB.AddPackages(ctx, pkgs...); err != nil {
				t.Fatalf("unexpected error adding packages: %s", err)
			}

			return nil
		},
	}

	if err := mgr.CreateDB(context.Background(), src); err != nil {
		t.Fatalf("unexpected error creating database with mock src: %s", err)
	}

	tallyDB, err := mgr.DB()
	if err != nil {
		t.Fatalf("unexpected error getting db: %s", err)
	}
	defer tallyDB.Close()

	for pkg, wantRepos := range wantPkgs {
		gotRepos, err := tallyDB.GetRepositories(context.Background(), pkg.System, pkg.Name)
		if err != nil {
			t.Fatalf("unexpected error getting repositories: %s", err)
		}
		if diff := cmp.Diff(wantRepos, gotRepos); diff != "" {
			t.Fatalf("unexpected repositories:\n%s", diff)
		}
	}

	metadata, err := mgr.Metadata()
	if err != nil {
		t.Fatalf("unexpected error getting metadata: %s", err)
	}
	if metadata == nil {
		t.Fatalf("unexpected nil metadata")
	}
}

func TestManagerCreateDB_UpdateError(t *testing.T) {
	dbDir, err := ioutil.TempDir("", "tally-db")
	if err != nil {
		t.Fatalf("unexpected error creating temp dir: %s", err)
	}
	defer os.RemoveAll(dbDir)

	mgr, err := NewManager(dbDir, ioutil.Discard)
	if err != nil {
		t.Fatalf("unexpected error creating manager: %s", err)
	}

	srcs := []Source{
		&mockSrc{
			name: "adds packages",
			update: func(ctx context.Context, tallyDB Writer) error {
				if err := tallyDB.AddPackages(ctx, []Package{
					{
						System:     "GO",
						Name:       "foobar",
						Repository: "github.com/foo/bar",
					},
					{
						System:     "NPM",
						Name:       "barfoo",
						Repository: "github.com/bar/foo",
					},
				}...); err != nil {
					t.Fatalf("unexpected error adding packages: %s", err)
				}

				return nil
			},
		},
		&mockSrc{
			name: "returns an error",
			update: func(ctx context.Context, tallyDB Writer) error {
				return fmt.Errorf("gerror")
			},
		},
		&mockSrc{
			name: "shouldn't be called",
			update: func(ctx context.Context, tallyDB Writer) error {
				t.Fatalf("unexpected call to third source in slice")
				return nil
			},
		},
	}

	if err := mgr.CreateDB(context.Background(), srcs...); err == nil {
		t.Fatalf("expected error creating database but got nil")
	}

	// The create operation should have aborted, so the metadata should be
	// nil
	metadata, err := mgr.Metadata()
	if err != nil {
		t.Fatalf("unexpected error getting database metadata: %s", err)
	}
	if metadata != nil {
		t.Fatalf("expected metadata to be nil but got: %+v", metadata)
	}
}

func TestManagerCreateDB_Overwrite(t *testing.T) {
	dbDir, err := ioutil.TempDir("", "tally-db")
	if err != nil {
		t.Fatalf("unexpected error creating temp dir: %s", err)
	}
	defer os.RemoveAll(dbDir)

	mgr, err := NewManager(dbDir, ioutil.Discard)
	if err != nil {
		t.Fatalf("unexpected error creating manager: %s", err)
	}
	if err := mgr.CreateDB(context.Background()); err != nil {
		t.Fatalf("unexpected error creating database: %s", err)
	}

	oldMeta, err := mgr.Metadata()
	if err != nil {
		t.Fatalf("unexpected error getting metadata: %s", err)
	}

	src := &mockSrc{
		name: "mock",
		update: func(ctx context.Context, tallyDB Writer) error {
			if err := tallyDB.AddPackages(ctx, []Package{
				{
					System:     "GO",
					Name:       "foobar",
					Repository: "github.com/foo/bar",
				},
				{
					System:     "NPM",
					Name:       "barfoo",
					Repository: "github.com/bar/foo",
				},
			}...); err != nil {
				t.Fatalf("unexpected error adding packages: %s", err)
			}

			return nil
		},
	}

	if err := mgr.CreateDB(context.Background(), src); err != nil {
		t.Fatalf("unexpected error creating database with mock src: %s", err)
	}

	// The metadata should have changed
	newMeta, err := mgr.Metadata()
	if err != nil {
		t.Fatalf("unexpected error getting new metadata: %s", err)
	}
	if newMeta.Equals(*oldMeta) {
		t.Fatalf("expected metadata of overwritten database to be different, but it's the same")
	}
}
