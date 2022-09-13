package local

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/db"
)

func TestAddGetScores(t *testing.T) {
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

	// There are no scores in the database, so we should get a not found
	// error
	if _, err := tallyDB.GetScores(context.Background(), "github.com/foo/bar"); err != db.ErrNotFound {
		t.Fatalf("expected error %q but got: %q", db.ErrNotFound, err)
	}

	scores := []db.Score{
		{
			Repository: "github.com/foo/bar",
			Score:      3.4,
		},
		// This score score should supersede the previous one.
		{
			Repository: "github.com/foo/bar",
			Score:      4.5,
		},
		{
			Repository: "github.com/bar/foo",
			Score:      8.4,
		},
		{
			Repository: "github.com/baz/foo",
			Score:      6.6,
		},
	}
	if err := tallyDB.AddScores(context.Background(), scores...); err != nil {
		t.Fatalf("unexpected error adding scores: %s", err)
	}

	want := []db.Score{
		{
			Repository: "github.com/bar/foo",
			Score:      8.4,
		},
		{
			Repository: "github.com/foo/bar",
			Score:      4.5,
		},
	}
	got, err := tallyDB.GetScores(context.Background(), "github.com/foo/bar", "github.com/bar/foo")
	if err != nil {
		t.Fatalf("unexpected error getting scores: %s", err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("unexpected scores:\n%s", diff)
	}
}
