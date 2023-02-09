package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/types"
)

func TestSqliteCachePutGet(t *testing.T) {
	tmpDir := t.TempDir()

	cache, err := NewSqliteCache(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error creating cache: %s", err)
	}

	repository := "github.com/foo/bar"

	wantScore := &types.Score{
		Score: 5.5,
	}

	if err := cache.PutScore(context.Background(), repository, wantScore); err != nil {
		t.Fatalf("unexpected error putting score in cache: %s", err)
	}

	gotScore, err := cache.GetScore(context.Background(), repository)
	if err != nil {
		t.Fatalf("unexpected error retrieving score from cache: %s", err)
	}

	if diff := cmp.Diff(wantScore, gotScore); diff != "" {
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
	wantScore := &types.Score{
		Score: 5.5,
	}
	if err := cache.PutScore(context.Background(), repository, wantScore); err != nil {
		t.Fatalf("unexpected error putting score in cache: %s", err)
	}
	gotScore, err := cache.GetScore(context.Background(), repository)
	if err != nil {
		t.Fatalf("unexpected error retrieving score from cache: %s", err)
	}
	if diff := cmp.Diff(wantScore, gotScore); diff != "" {
		t.Fatalf("unexpected score:\n%s", diff)
	}

	// Update the score
	wantScore = &types.Score{
		Score: 7.7,
	}
	if err := cache.PutScore(context.Background(), repository, wantScore); err != nil {
		t.Fatalf("unexpected error putting score in cache: %s", err)
	}
	gotScore, err = cache.GetScore(context.Background(), repository)
	if err != nil {
		t.Fatalf("unexpected error retrieving score from cache: %s", err)
	}
	if diff := cmp.Diff(wantScore, gotScore); diff != "" {
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
	if err := cache.PutScore(context.Background(), repository, &types.Score{
		Score: 5.5,
	}); err != nil {
		t.Fatalf("unexpected error putting score in cache: %s", err)
	}

	// override the time.Now function so that GetScore believes it's 2 hours
	// in the future
	cache.(*sqliteCache).timeNow = func() time.Time {
		return time.Now().Add(2 * time.Hour)
	}

	if _, err := cache.GetScore(context.Background(), repository); !errors.Is(err, ErrNotFound) {
		t.Errorf("unexpected error; wanted %q but got %q", ErrNotFound, err)
	}
}

func TestSqliteCacheGet_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	cache, err := NewSqliteCache(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error creating cache: %s", err)
	}

	if _, err := cache.GetScore(context.Background(), "github.com/foo/bar"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unexpected error; wanted %q but got %q", ErrNotFound, err)
	}
}
