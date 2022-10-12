package local

import (
	"fmt"
	"os"
	"path/filepath"
)

// Dir returns the default path to the local database
func Dir() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("getting user cache dir: %s", err)
	}

	return filepath.Join(cacheDir, "tally", "db"), nil
}
