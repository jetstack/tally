package cache

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jetstack/tally/internal/types"
	_ "modernc.org/sqlite"
)

const (
	createTableStatement = `
CREATE TABLE IF NOT EXISTS scores (
  repository text NOT NULL UNIQUE,
  score text NOT NULL,
  timestamp DATETIME NOT NULL
);
`

	selectScoreQuery = `
SELECT score, timestamp
FROM scores
WHERE repository = ?;
`

	insertScoreStatement = `
INSERT or REPLACE INTO scores
(repository, score, timestamp)
VALUES (?, ?, ?)
`
)

type sqliteCache struct {
	db      *sql.DB
	opts    *options
	timeNow func() time.Time
	mux     *sync.RWMutex
}

// NewSqliteCache returns a cache implementation that stores scorecard scores in a
// local sqlite database
func NewSqliteCache(dbDir string, opts ...Option) (Cache, error) {
	o := makeOptions(opts...)

	// If the directory isn't defined, then default to the users home cache
	// dir
	if dbDir == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			return nil, fmt.Errorf("getting user cache dir: %w", err)
		}
		dbDir = filepath.Join(cacheDir, "tally", "cache")
	}
	if err := os.MkdirAll(dbDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("creating cache directory: %w", err)
	}

	db, err := sql.Open("sqlite", fmt.Sprintf("%s/%s", dbDir, "cache.db"))
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(createTableStatement); err != nil {
		return nil, fmt.Errorf("creating scores table in database: %w", err)
	}

	return &sqliteCache{
		db:      db,
		opts:    o,
		mux:     &sync.RWMutex{},
		timeNow: time.Now,
	}, nil
}

// GetScore will retrieve a score from the cache
func (c *sqliteCache) GetScore(ctx context.Context, repository string) (*types.Score, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	rows, err := c.db.QueryContext(ctx, selectScoreQuery, repository)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("getting score from database: %w", err)
	}
	defer rows.Close()

	type row struct {
		Score     []byte
		Timestamp time.Time
	}
	var resp []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.Score, &r.Timestamp); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		resp = append(resp, r)
	}
	if len(resp) != 1 {
		return nil, ErrNotFound
	}

	if resp[0].Timestamp.Add(c.opts.Duration).Before(c.timeNow()) {
		return nil, ErrNotFound
	}

	score := &types.Score{}
	if err := json.Unmarshal(resp[0].Score, score); err != nil {
		return nil, fmt.Errorf("unmarshaling score from json: %w", err)
	}

	return score, nil
}

// PutScore will put a score into the cache
func (c *sqliteCache) PutScore(ctx context.Context, repository string, score *types.Score) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	scoreData, err := json.Marshal(score)
	if err != nil {
		return fmt.Errorf("marshaling score to JSON: %w", err)
	}
	if _, err := c.db.ExecContext(
		ctx,
		insertScoreStatement,
		repository,
		scoreData,
		c.timeNow(),
	); err != nil {
		return fmt.Errorf("inserting score: %w", err)
	}
	return nil
}
