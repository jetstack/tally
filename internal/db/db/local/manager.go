package local

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/jetstack/tally/internal/db"
)

// Manager manages the local database
type Manager interface {
	// CreateDB creates the database and populates it with data from the
	// provided sources.
	CreateDB(context.Context, ...db.Source) error

	// DB returns the database
	DB() (db.DB, error)

	// ExportDB exports the database as an Archive
	ExportDB() (Archive, error)

	// ImportDB imports the database from an Archive
	ImportDB(Archive) error

	// Metadata returns metadata about the database
	Metadata() (*Metadata, error)

	// NeedsUpdate returns true if the archive is different to the current
	// database
	NeedsUpdate(Archive) (bool, error)
}

type manager struct {
	dbDir string
}

// NewManager returns a manager for a local database
func NewManager(dbDir string) (Manager, error) {
	return &manager{
		dbDir: dbDir,
	}, nil
}

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
	if err := createDB(ctx, tempDBPath, srcs...); err != nil {
		return fmt.Errorf("creating database: %w", err)
	}

	// Create the metadata file
	h, err := sha256Sum(tempDBPath)
	if err != nil {
		return fmt.Errorf("generating checksum for database: %w", err)
	}
	metadata := &Metadata{
		Hash:      h,
		Timestamp: time.Now().UTC(),
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

func createDB(ctx context.Context, dbPath string, srcs ...db.Source) error {
	tallyDB, err := NewDB(dbPath, WithVacuumOnClose())
	if err != nil {
		return fmt.Errorf("creating database client: %w", err)
	}
	defer tallyDB.Close()

	// TODO: implement a logger for these prints
	fmt.Fprintf(os.Stderr, "Initializing database...\n")
	if err := tallyDB.Initialize(ctx); err != nil {
		return fmt.Errorf("initializing database: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Database initialized.\n")

	for _, src := range srcs {
		fmt.Fprintf(os.Stderr, "Populating database from source %q...\n", src)
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

func copyDB(src, dst string) error {
	if err := os.MkdirAll(dst, os.ModePerm); err != nil {
		return fmt.Errorf("making database directory: %w", err)
	}

	dsrc, err := os.Open(filepath.Join(src, "tally.db"))
	if err != nil {
		return fmt.Errorf("opening tally.db: %w", err)
	}
	defer dsrc.Close()

	msrc, err := os.Open(filepath.Join(src, "metadata.json"))
	if err != nil {
		return fmt.Errorf("opening metadata.json: %w", err)
	}
	defer msrc.Close()

	mdst, err := os.Create(filepath.Join(dst, "metadata.json"))
	if err != nil {
		return fmt.Errorf("opening metadata.json: %w", err)
	}
	defer mdst.Close()

	if _, err := io.Copy(mdst, msrc); err != nil {
		return fmt.Errorf("copying metadata.json: %w", err)
	}

	ddst, err := os.Create(filepath.Join(dst, "tally.db"))
	if err != nil {
		return fmt.Errorf("opening tally.db: %w", err)
	}
	defer ddst.Close()

	if _, err := io.Copy(ddst, dsrc); err != nil {
		return fmt.Errorf("copying tally.db: %w", err)
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
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// DB returns the managed database
func (m *manager) DB() (db.DB, error) {
	tallyDB, err := NewDB(filepath.Join(m.dbDir, "tally.db"))
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	return tallyDB, nil
}

// ExportDB exports the database as an Archive
func (m *manager) ExportDB() (Archive, error) {
	a, err := newDBArchive(m.dbDir)
	if err != nil {
		return nil, fmt.Errorf("creating archive: %w", err)
	}

	return a, nil
}

// ImportDB imports the database from an Archive
func (m *manager) ImportDB(a Archive) error {
	// Create a temporary directory
	tmpDir, err := ioutil.TempDir("", "tally-db-temp")
	if err != nil {
		return fmt.Errorf("getting temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write metadata to file
	metadata, err := a.Metadata()
	if err != nil {
		return fmt.Errorf("getting metadata: %w", err)
	}
	if err := createMetadataFile(filepath.Join(tmpDir, "metadata.json"), metadata); err != nil {
		return fmt.Errorf("creating metadata file: %w", err)
	}

	// Write database to file
	f, err := os.Create(filepath.Join(tmpDir, "tally.db"))
	if err != nil {
		return fmt.Errorf("creating database file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, a); err != nil {
		return fmt.Errorf("writing database to file: %w", err)
	}

	// Copy from the temporary directory to the database directory
	if err := copyDB(tmpDir, m.dbDir); err != nil {
		return fmt.Errorf("copying database: %w", err)
	}

	return nil
}

// Metadata returns metadata about the database
func (m *manager) Metadata() (*Metadata, error) {
	return metadataFromFile(filepath.Join(m.dbDir, "metadata.json"))
}

func metadataFromFile(path string) (*Metadata, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("statting database file: %w", err)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening metadata file: %w", err)
	}
	defer f.Close()

	metadata := &Metadata{}
	if err := json.NewDecoder(f).Decode(metadata); err != nil {
		return nil, fmt.Errorf("decoding metadata from file: %w", err)
	}

	return metadata, nil
}

// NeedsUpdate returns true if the archive is different to the current
// database
func (m *manager) NeedsUpdate(a Archive) (bool, error) {
	am, err := a.Metadata()
	if err != nil {
		return false, fmt.Errorf("getting metadata from archive: %w", err)
	}
	metadata, err := m.Metadata()
	if err != nil {
		return false, fmt.Errorf("getting database metadata: %w", err)
	}
	if metadata == nil {
		return true, nil
	}

	return !am.Equal(*metadata), nil
}
