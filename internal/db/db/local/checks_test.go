package local

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/db"
)

func TestAddGetChecks(t *testing.T) {
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

	// There are no checks in the database, so we should get a not found
	// error
	if _, err := tallyDB.GetChecks(context.Background(), "github.com/foo/bar"); err != db.ErrNotFound {
		t.Fatalf("expected error %q but got: %q", db.ErrNotFound, err)
	}

	checks := []db.Check{
		{
			Name:       "foo",
			Repository: "github.com/foo/bar",
			Score:      3,
		},
		// This check score should supersede the previous one.
		{
			Name:       "foo",
			Repository: "github.com/foo/bar",
			Score:      4,
		},
		{
			Name:       "foo",
			Repository: "github.com/bar/foo",
			Score:      8,
		},
		{
			Name:       "bar",
			Repository: "github.com/bar/foo",
			Score:      7,
		},
		{
			Name:       "bar",
			Repository: "github.com/foo/bar",
			Score:      2,
		},
		{
			Name:       "baz",
			Repository: "github.com/baz/foo",
			Score:      2,
		},
	}

	if err := tallyDB.AddChecks(context.Background(), checks...); err != nil {
		t.Fatalf("unexpected error adding checks: %s", err)
	}

	wantChecks := []db.Check{
		{
			Name:       "bar",
			Repository: "github.com/foo/bar",
			Score:      2,
		},
		{
			Name:       "baz",
			Repository: "github.com/foo/bar",
			Score:      0,
		},
		{
			Name:       "foo",
			Repository: "github.com/foo/bar",
			Score:      4,
		},
	}

	gotChecks, err := tallyDB.GetChecks(context.Background(), "github.com/foo/bar")
	if err != nil {
		t.Fatalf("unexpected error getting checks: %s", err)
	}

	if diff := cmp.Diff(wantChecks, gotChecks); diff != "" {
		t.Fatalf("unexpected checks:\n%s", diff)
	}
}
