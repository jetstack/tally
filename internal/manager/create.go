package manager

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jetstack/tally/internal/db"
	"github.com/jetstack/tally/internal/db/local"
)

// CreateDB creates the database and populates it with data from the provided
// sources.
func (m *manager) CreateDB(ctx context.Context, srcs ...db.Source) error {
	tempDir, err := ioutil.TempDir("", "tally-db")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	tempDBPath := filepath.Join(tempDir, "tally.db")
	tempMetadataPath := filepath.Join(tempDir, "metadata.json")

	// Create the database
	if err := m.createDB(ctx, tempDBPath, srcs...); err != nil {
		return fmt.Errorf("creating database: %w", err)
	}

	// Create the metadata file
	h, err := sha256Sum(tempDBPath)
	if err != nil {
		return fmt.Errorf("generating checksum for database: %w", err)
	}
	metadata := &Metadata{
		SHA256: h,
	}
	if err := createMetadataFile(tempMetadataPath, metadata); err != nil {
		return fmt.Errorf("creating metadata file: %w", err)
	}

	// Copy the database and the metadata file to the managed dir
	if err := copyDB(tempDir, m.dbDir); err != nil {
		return fmt.Errorf("copying database: %w", err)
	}

	return nil
}

func (m *manager) createDB(ctx context.Context, dbPath string, srcs ...db.Source) error {
	tallyDB, err := local.NewDB(dbPath, local.WithVacuumOnClose())
	if err != nil {
		return fmt.Errorf("creating database client: %w", err)
	}
	defer tallyDB.Close()

	fmt.Fprintf(m.w, "Initializing database...\n")
	if err := tallyDB.Initialize(ctx); err != nil {
		return fmt.Errorf("initializing database: %w", err)
	}
	fmt.Fprintf(m.w, "Database initialized.\n")

	for _, src := range srcs {
		fmt.Fprintf(m.w, "Populating database from source %q...\n", src)
		if err := src.Update(ctx, tallyDB); err != nil {
			return fmt.Errorf("populating database from source: %q: %w", src, err)
		}
	}

	return nil
}

func createMetadataFile(path string, metadata *Metadata) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating metadata file: %w", err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(metadata); err != nil {
		return fmt.Errorf("writing metadata file: %w", err)
	}

	return nil
}

func sha256Sum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hashing file: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func createFile(name string, r io.Reader) error {
	f, err := os.Create(name)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("copying data to file: %w", err)
	}

	return nil
}
