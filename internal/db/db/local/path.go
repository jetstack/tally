package local

import (
	"fmt"
	"os"
	"path/filepath"
)

// Path returns the default path to the store
func Path() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("getting user cache dir: %s", err)
	}

	return filepath.Join(cacheDir, "tally", "db", "tally.db"), nil
}
