package local

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/v1/random"
)

func TestImportDB_RandomImage(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "tally-db-temp")
	if err != nil {
		t.Fatalf("unexpected error creating temporary directory: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	img, err := random.Image(1024, 2)
	if err != nil {
		t.Fatalf("unexpected error creating random image: %s", err)
	}

	importPath := filepath.Join(tmpDir, "tally.db")
	if err := ImportDB(img, importPath); err == nil {
		t.Fatalf("expected error but got nil")
	}
}
