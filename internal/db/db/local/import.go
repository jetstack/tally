package local

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// ImportDB imports the database from img to the dbPath
func ImportDB(img v1.Image, dbPath string) error {
	// Import the database to a temporary file
	tmpDir, err := ioutil.TempDir("", "tally-db-temp")
	if err != nil {
		return fmt.Errorf("getting temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpPath := filepath.Join(tmpDir, "tally.db")

	if err := importDB(img, tmpPath); err != nil {
		return fmt.Errorf("importing database to temp file: %w", err)
	}

	// Copy the temporary file to the actual path
	f, err := os.Create(dbPath)
	if err != nil {
		return fmt.Errorf("opening file %s: %w", dbPath, err)
	}

	tf, err := os.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("opening file %s: %w", tmpPath, err)
	}

	if _, err := io.Copy(f, tf); err != nil {
		return fmt.Errorf("copying temp file to db path: %w", err)
	}

	return nil
}

func importDB(img v1.Image, dbPath string) error {
	mt, err := img.MediaType()
	if err != nil {
		return fmt.Errorf("getting image media type: %w", err)
	}
	if mt != types.OCIManifestSchema1 {
		return fmt.Errorf("unexpected media type: %s", mt)
	}

	manifest, err := img.Manifest()
	if err != nil {
		return fmt.Errorf("getting image manifest: %w", err)
	}
	if manifest.Config.MediaType != types.OCIConfigJSON {
		return fmt.Errorf("unexpected config media type: %s", manifest.Config.MediaType)
	}

	layers, err := img.Layers()
	if err != nil {
		return fmt.Errorf("getting layers from image: %w", err)
	}

	if len(layers) != 1 {
		return fmt.Errorf("unexpected number of layers: %d", len(layers))
	}

	mt, err = layers[0].MediaType()
	if err != nil {
		return fmt.Errorf("getting layer media type: %w", err)
	}
	if mt != LayerMediaType {
		return fmt.Errorf("unexpected layer media type: %s", mt)
	}

	rc, err := layers[0].Uncompressed()
	if err != nil {
		return fmt.Errorf("getting uncompressed layer: %w", err)
	}
	defer rc.Close()

	tr := tar.NewReader(rc)

	f, err := os.Create(dbPath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("extracting layer: %w", err)
		}

		if header.Name != "tally.db" {
			return fmt.Errorf("unexpected file in layer: %s", header.Name)
		}

		if _, err := io.Copy(f, tr); err != nil {
			return fmt.Errorf("copying database to file: %w", err)
		}
	}

	return nil
}
