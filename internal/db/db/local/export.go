package local

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// LayerMediaType is the media type of the layer in the exported database
const LayerMediaType = types.MediaType("application/vnd.jetstack.tally.db.layer.v1+gzip")

// ExportDB exports the database as an OCI image
func ExportDB(dbPath string) (v1.Image, error) {
	layer, err := layerFromPath(dbPath)
	if err != nil {
		return nil, fmt.Errorf("creating layer: %w", err)
	}

	img, err := mutate.AppendLayers(
		mutate.MediaType(
			mutate.ConfigMediaType(
				empty.Image,
				types.OCIConfigJSON,
			),
			types.OCIManifestSchema1,
		),
		layer,
	)
	if err != nil {
		return nil, fmt.Errorf("creating image: %w", err)
	}

	return img, nil
}

func layerFromPath(dbPath string) (v1.Layer, error) {
	f, err := os.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening path: %w", err)
	}
	defer f.Close()

	finfo, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("statting file: %w", err)
	}

	b := &bytes.Buffer{}
	w := tar.NewWriter(b)

	if err := w.WriteHeader(&tar.Header{
		Name: "tally.db",
		Size: finfo.Size(),
	}); err != nil {
		return nil, fmt.Errorf("writing tar header: %w", err)
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return nil, fmt.Errorf("writing data to tar archive: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("closing tar writer: %w", err)
	}

	return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewBuffer(b.Bytes())), nil
	}, tarball.WithMediaType(LayerMediaType))

}
