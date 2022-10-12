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

	"github.com/jetstack/tally/internal/db"
	localdb "github.com/jetstack/tally/internal/db/local/db"
)

// Option is a functional option that configures the manager
type Option func(mgr *manager)

// WithDir is a functional option that configures the local database directory
func WithDir(dbDir string) Option {
	return func(mgr *manager) {
		mgr.dbDir = dbDir
	}
}

// WithWriter is a functional option that configures an io.Writer that the
// manager will write output to
func WithWriter(w io.Writer) Option {
	return func(mgr *manager) {
		mgr.w = w
	}
}

// Manager manages the local database
type Manager interface {
	// CreateDB creates the database from the provided sources
	CreateDB(context.Context, ...db.Source) error

	// DB returns the managed database
	DB() (db.DB, error)

	// UpdateDB updates the database from the provided archive. Returns true
	// if the database was updated and false if the database was already at
	// the provided version.
	UpdateDB(Archive) (bool, error)

	// Metadata returns metadata about the current database
	Metadata() (*Metadata, error)
}

type manager struct {
	dbDir string
	w     io.Writer
}

// NewManager returns a new manager that manages the local database
func NewManager(opts ...Option) (Manager, error) {
	mgr := &manager{
		w: io.Discard,
	}
	for _, opt := range opts {
		opt(mgr)
	}

	if mgr.dbDir == "" {
		dbDir, err := Dir()
		if err != nil {
			return nil, fmt.Errorf("getting database directory: %w", err)
		}
		mgr.dbDir = dbDir
	}

	return mgr, nil
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
	tallyDB, err := localdb.NewDB(dbPath, localdb.WithVacuumOnClose())
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
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// DB returns the managed database
func (m *manager) DB() (db.DB, error) {
	tallyDB, err := localdb.NewDB(filepath.Join(m.dbDir, "tally.db"))
	if err != nil {
		return nil, fmt.Errorf("getting database: %w", err)
	}

	return tallyDB, nil
}

// UpdateDB updates the database from the provided archive
func (m *manager) UpdateDB(a Archive) (bool, error) {
	update, err := m.needsUpdate(a)
	if err != nil {
		return false, fmt.Errorf("checking if update is required: %w", err)
	}
	if !update {
		return false, nil
	}

	if err := importDB(m.dbDir, a); err != nil {
		return false, fmt.Errorf("extracting archive: %w", err)
	}

	return true, nil
}

func importDB(dbDir string, a Archive) error {
	tmpDir, err := ioutil.TempDir("", "tally")
	if err != nil {
		return fmt.Errorf("creating temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	metadata, err := a.Metadata()
	if err != nil {
		return fmt.Errorf("getting metadata from archive: %w", err)
	}
	if err := createMetadataFile(filepath.Join(tmpDir, "metadata.json"), metadata); err != nil {
		return fmt.Errorf("getting metadata from archive: %w", err)
	}

	data, err := a.DB()
	if err != nil {
		return fmt.Errorf("getting database data from archive: %w", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "tally.db"), data, os.ModePerm); err != nil {
		return fmt.Errorf("writing database file: %w", err)
	}

	if err := copyDB(tmpDir, dbDir); err != nil {
		return fmt.Errorf("copying database to directory: %w", err)
	}

	return nil
}

func (m *manager) needsUpdate(a Archive) (bool, error) {
	am, err := a.Metadata()
	if err != nil {
		return false, fmt.Errorf("getting archive metadata: %w", err)
	}
	if am == nil {
		return false, nil
	}

	dm, err := m.Metadata()
	if err != nil {
		return false, fmt.Errorf("getting database metadata: %w", err)
	}
	if dm == nil {
		return true, nil
	}

	return !am.Equals(*dm), nil

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

// Metadata returns metadata about the current database
func (m *manager) Metadata() (*Metadata, error) {
	f, err := os.Open(filepath.Join(m.dbDir, "metadata.json"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("opening metadata file: %w", err)
	}
	defer f.Close()

	metadata := &Metadata{}
	if err := json.NewDecoder(f).Decode(metadata); err != nil {
		return nil, fmt.Errorf("decoding metadata: %w", err)
	}

	return metadata, nil
}
