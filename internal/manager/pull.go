package manager

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cheggaaa/pb/v3"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

const (
	dbMediaType       = types.MediaType("application/vnd.jetstack.tally.db.layer.v1+gzip")
	metadataMediaType = types.MediaType("application/vnd.jetstack.tally.metadata.layer.v1")
)

// PullDB pulls the database from a remote reference to the managed directory.
// Returns a bool that indicates whether the database is already up-to-date.
func (m *manager) PullDB(ctx context.Context, tag string) (bool, error) {
	// Get remote image
	ref, err := name.ParseReference(tag)
	if err != nil {
		return false, fmt.Errorf("parsing reference: %w", err)
	}
	img, err := remote.Image(ref, remote.WithContext(ctx), remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return false, fmt.Errorf("fetching image: %w", err)
	}

	// Get metadata from image, check if the database is up-to-date
	imeta, err := extractMetadata(img)
	if err != nil {
		return false, fmt.Errorf("getting metadata from image: %w", err)
	}
	meta, err := m.Metadata()
	if err != nil {
		return false, fmt.Errorf("getting database metadata: %w", err)
	}
	if meta != nil && meta.Equals(*imeta) {
		return false, nil
	}

	// Write database and metadata to a temp dir
	tmpDir, err := ioutil.TempDir("", "tally")
	if err != nil {
		return false, fmt.Errorf("creating temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	if err := extractDB(img, tmpDir, m.w); err != nil {
		return false, fmt.Errorf("extracting database from image: %w", err)
	}
	if err := createMetadataFile(filepath.Join(tmpDir, "metadata.json"), imeta); err != nil {
		return false, fmt.Errorf("creating metadata file: %w", err)
	}

	// Copy temp dir to local directory
	if err := copyDB(tmpDir, m.dbDir); err != nil {
		return false, fmt.Errorf("copying database: %w", err)
	}

	return true, nil
}

func extractMetadata(img v1.Image) (*Metadata, error) {
	layers, err := img.Layers()
	if err != nil {
		return nil, fmt.Errorf("getting layers: %w", err)
	}
	if len(layers) != 2 {
		return nil, fmt.Errorf("unexpected number of layers in image, wanted %d but got %d", 2, len(layers))
	}
	layer := layers[0]

	mt, err := layer.MediaType()
	if err != nil {
		return nil, fmt.Errorf("getting layer media type: %w", err)
	}

	if mt != metadataMediaType {
		return nil, fmt.Errorf("unexpected media type: %s", mt)
	}

	// The metadata file is not compressed, so this just returns the actual
	// data
	rc, err := layer.Compressed()
	if err != nil {
		return nil, fmt.Errorf("getting uncompressed layer: %w", err)
	}
	defer rc.Close()

	metadata := &Metadata{}
	if err := json.NewDecoder(rc).Decode(metadata); err != nil {
		return nil, fmt.Errorf("decoding metadata json: %w", err)
	}

	return metadata, nil
}

func extractDB(img v1.Image, dir string, w io.Writer) error {
	layers, err := img.Layers()
	if err != nil {
		return fmt.Errorf("getting layers: %w", err)
	}
	if len(layers) != 2 {
		return fmt.Errorf("unexpected number of layers in image, wanted %d but got %d", 2, len(layers))
	}
	layer := layers[1]

	mt, err := layer.MediaType()
	if err != nil {
		return fmt.Errorf("getting layer media type: %w", err)
	}
	if mt != dbMediaType {
		return fmt.Errorf("unexpected media type: %w", err)
	}

	size, err := layer.Size()
	if err != nil {
		return fmt.Errorf("getting layer size: %w", err)
	}

	f, err := os.Create(filepath.Join(dir, "tally.db"))
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	rc, err := layer.Compressed()
	if err != nil {
		return fmt.Errorf("getting compressed layer: %w", err)
	}
	defer rc.Close()

	bar := pb.Full.Start64(size)
	bar.SetWriter(w)
	defer bar.Finish()

	gr, err := gzip.NewReader(bar.NewProxyReader(rc))
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gr.Close()

	if _, err := io.Copy(f, gr); err != nil {
		return fmt.Errorf("extracting database: %w", err)
	}

	return nil
}

func copyDB(src, dst string) error {
	if err := os.MkdirAll(dst, os.ModePerm); err != nil {
		return fmt.Errorf("making database directory: %w", err)
	}

	files := []string{
		"tally.db",
		"metadata.json",
	}
	for _, f := range files {
		src, err := os.Open(filepath.Join(src, f))
		if err != nil {
			return fmt.Errorf("opening %s: %w", f, err)
		}
		defer src.Close()

		dst, err := os.Create(filepath.Join(dst, f))
		if err != nil {
			return fmt.Errorf("creating %s: %w", f, err)
		}
		defer dst.Close()

		if _, err := io.Copy(dst, src); err != nil {
			return fmt.Errorf("copying %s: %w", f, err)
		}
	}

	return nil
}
