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

	"github.com/ossf/scorecard-webapp/app/generated/models"
	_ "modernc.org/sqlite"
)

const (
	createTableStatement = `
CREATE TABLE IF NOT EXISTS results (
  repository text NOT NULL UNIQUE,
  result text NOT NULL,
  timestamp DATETIME NOT NULL
);
`

	selectResultQuery = `
SELECT result, timestamp
FROM results
WHERE repository = ?;
`

	insertResultStatement = `
INSERT or REPLACE INTO results
(repository, result, timestamp)
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

// GetResult will retrieve a scorecard result from the cache
func (c *sqliteCache) GetResult(ctx context.Context, repository string) (*models.ScorecardResult, error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	rows, err := c.db.QueryContext(ctx, selectResultQuery, repository)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("getting score from database: %w", err)
	}
	defer rows.Close()

	type row struct {
		Result    []byte
		Timestamp time.Time
	}
	var resp []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.Result, &r.Timestamp); err != nil {
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

	result := &models.ScorecardResult{}
	if err := json.Unmarshal(resp[0].Result, result); err != nil {
		return nil, fmt.Errorf("unmarshaling score from json: %w", err)
	}

	return result, nil
}

// PutResult will put a scorecard result into the cache
func (c *sqliteCache) PutResult(ctx context.Context, repository string, result *models.ScorecardResult) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	scoreData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshaling score to JSON: %w", err)
	}
	if _, err := c.db.ExecContext(
		ctx,
		insertResultStatement,
		repository,
		scoreData,
		c.timeNow(),
	); err != nil {
		return fmt.Errorf("inserting score: %w", err)
	}
	return nil
}
