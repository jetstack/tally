package local

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestExportImportDB(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "tally-db-test")
	if err != nil {
		t.Fatalf("unexpected error creating temp dir: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	exportPath := filepath.Join(tmpDir, "tally.db")

	// Test that we get an error when the database doesn't exist
	if _, err := ExportDB(exportPath); err == nil {
		t.Fatalf("expected error exporting non-existent file but got nil")
	}

	// Initialize the database
	tallyDB, err := NewDB(exportPath)
	if err != nil {
		t.Fatalf("unexpected error opening database: %s", err)
	}
	if err := tallyDB.Initialize(context.Background()); err != nil {
		t.Fatalf("unexpected error intializing database: %s", err)
	}
	if err := tallyDB.Close(); err != nil {
		t.Fatalf("unexpected error closing database: %s", err)
	}

	// Export the database
	img, err := ExportDB(exportPath)
	if err != nil {
		t.Fatalf("unexpected error exporting database: %s", err)
	}

	// Import the database to another path
	importPath := filepath.Join(tmpDir, "import.db")
	if err := ImportDB(img, importPath); err != nil {
		t.Fatalf("unexpected error importing database: %s", err)
	}

	// Check that the db we imported is exactly the same as the one we
	// exported
	exportSha, err := sha256Sum(exportPath)
	if err != nil {
		t.Fatalf("unexpected error getting sha256sum of exported db: %s", err)
	}
	importSha, err := sha256Sum(importPath)
	if err != nil {
		t.Fatalf("unexpected error getting sha256sum of imported db: %s", err)
	}
	if exportSha != importSha {
		t.Fatalf("imported sha256 doesn't match exported sha256, got %s but want %s", exportSha, importSha)
	}
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
