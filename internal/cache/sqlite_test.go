package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ossf/scorecard-webapp/app/generated/models"
)

func TestSqliteCachePutGet(t *testing.T) {
	tmpDir := t.TempDir()

	cache, err := NewSqliteCache(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error creating cache: %s", err)
	}

	repository := "github.com/foo/bar"

	wantScorecardResult := &models.ScorecardResult{
		Score: 5.5,
	}

	if err := cache.PutResult(context.Background(), repository, wantScorecardResult); err != nil {
		t.Fatalf("unexpected error putting score in cache: %s", err)
	}

	gotScorecardResult, err := cache.GetResult(context.Background(), repository)
	if err != nil {
		t.Fatalf("unexpected error retrieving score from cache: %s", err)
	}

	if diff := cmp.Diff(wantScorecardResult, gotScorecardResult); diff != "" {
		t.Fatalf("unexpected score:\n%s", diff)
	}
}

func TestSqliteCachePutGet_Replace(t *testing.T) {
	tmpDir := t.TempDir()

	cache, err := NewSqliteCache(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error creating cache: %s", err)
	}

	repository := "github.com/foo/bar"

	// Put the first score for the repository in
	wantScorecardResult := &models.ScorecardResult{
		Score: 5.5,
	}
	if err := cache.PutResult(context.Background(), repository, wantScorecardResult); err != nil {
		t.Fatalf("unexpected error putting score in cache: %s", err)
	}
	gotScorecardResult, err := cache.GetResult(context.Background(), repository)
	if err != nil {
		t.Fatalf("unexpected error retrieving score from cache: %s", err)
	}
	if diff := cmp.Diff(wantScorecardResult, gotScorecardResult); diff != "" {
		t.Fatalf("unexpected score:\n%s", diff)
	}

	// Update the score
	wantScorecardResult = &models.ScorecardResult{
		Score: 7.7,
	}
	if err := cache.PutResult(context.Background(), repository, wantScorecardResult); err != nil {
		t.Fatalf("unexpected error putting score in cache: %s", err)
	}
	gotScorecardResult, err = cache.GetResult(context.Background(), repository)
	if err != nil {
		t.Fatalf("unexpected error retrieving score from cache: %s", err)
	}
	if diff := cmp.Diff(wantScorecardResult, gotScorecardResult); diff != "" {
		t.Fatalf("unexpected score:\n%s", diff)
	}
}

func TestSqliteCachePutGet_Expired(t *testing.T) {
	tmpDir := t.TempDir()

	cache, err := NewSqliteCache(tmpDir, WithDuration(1*time.Minute))
	if err != nil {
		t.Fatalf("unexpected error creating cache: %s", err)
	}

	repository := "github.com/foo/bar"
	if err := cache.PutResult(context.Background(), repository, &models.ScorecardResult{
		Score: 5.5,
	}); err != nil {
		t.Fatalf("unexpected error putting score in cache: %s", err)
	}

	// override the time.Now function so that GetResult believes it's 2 hours
	// in the future
	cache.(*sqliteCache).timeNow = func() time.Time {
		return time.Now().Add(2 * time.Hour)
	}

	if _, err := cache.GetResult(context.Background(), repository); !errors.Is(err, ErrNotFound) {
		t.Errorf("unexpected error; wanted %q but got %q", ErrNotFound, err)
	}
}

func TestSqliteCacheGet_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	cache, err := NewSqliteCache(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error creating cache: %s", err)
	}

	if _, err := cache.GetResult(context.Background(), "github.com/foo/bar"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unexpected error; wanted %q but got %q", ErrNotFound, err)
	}
}
