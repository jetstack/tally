package db

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAddPackagesGetRepositories(t *testing.T) {
	tmpDB, err := ioutil.TempFile("", "tally-db-test")
	if err != nil {
		t.Fatalf("unexpected error creating temp database: %s", err)
	}
	defer os.Remove(tmpDB.Name())

	tallyDB, err := NewDB(tmpDB.Name())
	if err != nil {
		t.Fatalf("unexpected error opening database: %s", err)
	}
	defer tallyDB.Close()

	if err := tallyDB.Initialize(context.Background()); err != nil {
		t.Fatalf("unexpected error intializing database: %s", err)
	}

	// There are no packages in the database, so we should get a not found
	// error
	if _, err := tallyDB.GetRepositories(context.Background(), "golang", "github.com/foo/bar"); err != ErrNotFound {
		t.Fatalf("expected error %q but got: %q", ErrNotFound, err)
	}

	packages := []Package{
		{
			Type:       "golang",
			Name:       "github.com/foo/bar",
			Repository: "github.com/foo/bar",
		},
		{
			Type:       "golang",
			Name:       "github.com/foo/bar",
			Repository: "github.com/foo/bar-foo",
		},
		{
			Type:       "npm",
			Name:       "foo",
			Repository: "github.com/bar/foo",
		},
		{
			Type:       "cargo",
			Name:       "bar",
			Repository: "github.com/foo/bar",
		},
	}

	if err := tallyDB.AddPackages(context.Background(), packages...); err != nil {
		t.Fatalf("unexpected error adding packages: %s", err)
	}

	want := []string{
		"github.com/foo/bar",
		"github.com/foo/bar-foo",
	}
	got, err := tallyDB.GetRepositories(context.Background(), "golang", "github.com/foo/bar")
	if err != nil {
		t.Fatalf("unexpected error getting packages: %s", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("unexpected repositories:\n%s", diff)
	}

	want2 := []string{
		"github.com/foo/bar",
	}
	got2, err := tallyDB.GetRepositories(context.Background(), "cargo", "bar")
	if err != nil {
		t.Fatalf("unexpected error getting packages: %s", err)
	}

	if diff := cmp.Diff(want2, got2); diff != "" {
		t.Fatalf("unexpected repositories:\n%s", diff)
	}
}
