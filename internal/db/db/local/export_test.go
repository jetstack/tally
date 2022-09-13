package local

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/v1/types"
)

func TestExportDB(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "tally-db-test")
	if err != nil {
		t.Fatalf("unexpected error creating temp dir: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "tally.db")

	// Test that we get an error when the database doesn't exist
	if _, err := ExportDB(dbPath); err == nil {
		t.Fatalf("expected error exporting non-existent file but got nil")
	}

	// Initialize the database
	tallyDB, err := NewDB(dbPath)
	if err != nil {
		t.Fatalf("unexpected error opening database: %s", err)
	}

	if err := tallyDB.Initialize(context.Background()); err != nil {
		t.Fatalf("unexpected error intializing database: %s", err)
	}

	if err := tallyDB.Close(); err != nil {
		t.Fatalf("unexpected error closing database: %s", err)
	}

	// Get the sha256 of the database
	tmpDB, err := os.Open(dbPath)
	if err != nil {
		t.Fatalf("unexpected error opening database: %s", err)
	}
	wantSha, err := sha256Sum(tmpDB)
	if err != nil {
		t.Fatalf("unexpected error getting sha256 from database file: %s", err)
	}
	if err := tmpDB.Close(); err != nil {
		t.Fatalf("unexpected error closing temp file: %s", err)
	}

	// Export the database
	img, err := ExportDB(dbPath)
	if err != nil {
		t.Fatalf("unexpected error exporting database: %s", err)
	}

	// The media type should be application/vnd.oci.image.manifest.v1+json
	gotMT, err := img.MediaType()
	if err != nil {
		t.Fatalf("unexpected error getting image media type: %s", err)
	}
	wantMT := types.OCIManifestSchema1
	if gotMT != wantMT {
		t.Fatalf("expected media type %q but got %q", wantMT, gotMT)
	}

	// The config media type should be application/vnd.oci.image.config.v1+json
	manifest, err := img.Manifest()
	if err != nil {
		t.Fatalf("unexpected error getting image manifest: %s", err)
	}
	gotCfgMT := manifest.Config.MediaType
	wantCfgMT := types.OCIConfigJSON
	if gotCfgMT != wantCfgMT {
		t.Fatalf("expected config media type %q but got %q", wantCfgMT, gotCfgMT)
	}

	// There should be 1 layer
	layers, err := img.Layers()
	if err != nil {
		t.Fatalf("unexpected error getting layers: %s", err)
	}
	if len(layers) != 1 {
		t.Fatalf("expected %d layers but got %d", 1, len(layers))
	}

	// The layer media type should be LayerMediaType
	gotLayerMT, err := layers[0].MediaType()
	if err != nil {
		t.Fatalf("unexpected error getting layer media type: %s", err)
	}
	if gotLayerMT != LayerMediaType {
		t.Fatalf("expected layer media type %q but got %q", LayerMediaType, gotLayerMT)
	}

	// The layer should be a tar containing one file 'tally.db'. The sha256
	// of that file should match the sha256 of the database we exported.
	rc, err := layers[0].Uncompressed()
	if err != nil {
		t.Fatalf("unexpected error getting layer tar: %s", err)
	}
	defer rc.Close()

	tr := tar.NewReader(rc)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error iterating over layer tar: %s", err)
		}

		if header.Name != "tally.db" {
			t.Fatalf("unexpected header name in layer tar: %s", header.Name)
		}

		gotSha, err := sha256Sum(tr)
		if err != nil {
			t.Fatalf("unexpected error getting sha256 from db in layer tar: %s", err)
		}

		if gotSha != wantSha {
			t.Fatalf("unexpected sha256 for file in layer, want %q but got %q", wantSha, gotSha)
		}
	}
}

func sha256Sum(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
