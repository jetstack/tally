package local

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

const (
	// layerMediaType is the media type of the layer in the exported database
	layerMediaType = types.MediaType("application/vnd.jetstack.tally.db.layer.v1+gzip")

	// annotationHash is the image annotation that stores the db hash
	annotationHash = "io.jetstack.tally/hash"

	// annotationTimestamp is the image annotation that stores the db
	// timestamp
	annotationTimestamp = "io.jetstack.tally/timestamp"
)

// Archive is an archive containing the database. Call Read to read the database
// contents from the archive and Close to close it. Metadata returns metadata
// about the database.
type Archive interface {
	io.ReadCloser
	Metadata() (*Metadata, error)
}

// WriteArchiveToRemote writes an archive to a remote registry
func WriteArchiveToRemote(ref name.Reference, archive Archive, remoteOpts ...remote.Option) (name.Digest, error) {
	metadata, err := archive.Metadata()
	if err != nil {
		return name.Digest{}, fmt.Errorf("getting metadata: %w", err)
	}

	layer, err := makeDBLayer(archive)
	if err != nil {
		return name.Digest{}, fmt.Errorf("creating layer: %w", err)
	}

	img, err := mutate.AppendLayers(
		mutate.Annotations(mutate.MediaType(
			mutate.ConfigMediaType(
				empty.Image,
				types.OCIConfigJSON,
			),
			types.OCIManifestSchema1,
		), map[string]string{
			annotationHash:      metadata.Hash,
			annotationTimestamp: metadata.Timestamp.Format(time.RFC3339Nano),
		}).(v1.Image),
		layer,
	)
	if err != nil {
		return name.Digest{}, fmt.Errorf("creating image: %w", err)
	}

	digest, err := img.Digest()
	if err != nil {
		return name.Digest{}, fmt.Errorf("getting image digest: %w", err)
	}

	if err := remote.Write(ref, img, remoteOpts...); err != nil {
		return name.Digest{}, fmt.Errorf("writing remote image: %w", err)
	}

	return ref.Context().Digest(digest.String()), nil
}

func makeDBLayer(r io.Reader) (v1.Layer, error) {
	b := &bytes.Buffer{}
	w := tar.NewWriter(b)

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	if err := w.WriteHeader(&tar.Header{
		Name: "tally.db",
		Size: int64(len(data)),
	}); err != nil {
		return nil, fmt.Errorf("writing tar header: %w", err)
	}

	if _, err := w.Write(data); err != nil {
		return nil, fmt.Errorf("writing data to tar archive: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("closing tar writer: %w", err)
	}

	return tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return ioutil.NopCloser(bytes.NewBuffer(b.Bytes())), nil
	}, tarball.WithMediaType(layerMediaType))

}

// GetArchiveFromRemote retrieves an archive from a remote registry
func GetArchiveFromRemote(ref name.Reference, remoteOpts ...remote.Option) (Archive, error) {
	desc, err := remote.Get(ref, remoteOpts...)
	if err != nil {
		return nil, fmt.Errorf("getting remote descriptor: %w", err)
	}

	img, err := desc.Image()
	if err != nil {
		return nil, fmt.Errorf("getting image from descriptor: %w", err)
	}

	return newImgArchive(img)
}

type imgArchive struct {
	io.Closer
	io.Reader
	metadata *Metadata
}

func newImgArchive(img v1.Image) (Archive, error) {
	if err := validateImage(img); err != nil {
		return nil, fmt.Errorf("validating image: %w", err)
	}

	metadata, err := imageMetadata(img)
	if err != nil {
		return nil, fmt.Errorf("getting metadata: %w", err)
	}

	layers, err := img.Layers()
	if err != nil {
		return nil, fmt.Errorf("getting layers from image: %w", err)
	}

	rc, err := layers[0].Uncompressed()
	if err != nil {
		return nil, fmt.Errorf("getting uncompressed layer: %w", err)
	}

	tr := tar.NewReader(rc)

	ia := &imgArchive{
		Closer:   rc,
		Reader:   tr,
		metadata: metadata,
	}

	for {
		header, err := tr.Next()
		if err != nil {
			break
		}

		if header.Name == "tally.db" {
			return ia, nil
		}
	}

	ia.Close()

	return nil, fmt.Errorf("couldn't find database in tar")
}

func validateImage(img v1.Image) error {
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
	if cmt := manifest.Config.MediaType; cmt != types.OCIConfigJSON {
		return fmt.Errorf("unexpected config media type: %s", cmt)
	}

	layers, err := img.Layers()
	if err != nil {
		return fmt.Errorf("getting layers from image: %w", err)
	}
	if len(layers) != 1 {
		return fmt.Errorf("unexpected number of layers: %d", len(layers))
	}
	lmt, err := layers[0].MediaType()
	if err != nil {
		return fmt.Errorf("getting media type from layer: %w", err)
	}
	if lmt != layerMediaType {
		return fmt.Errorf("unexpected layer media type: %s", lmt)
	}

	return nil
}

func imageMetadata(img v1.Image) (*Metadata, error) {
	manifest, err := img.Manifest()
	if err != nil {
		return nil, fmt.Errorf("getting image manifest: %w", err)
	}

	h, ok := manifest.Annotations[annotationHash]
	if !ok || h == "" {
		return nil, fmt.Errorf("couldn't find hash in image annotations")
	}

	t, ok := manifest.Annotations[annotationTimestamp]
	if !ok || t == "" {
		return nil, fmt.Errorf("couldn't find timestamp in image annotations")
	}
	ts, err := time.Parse(time.RFC3339Nano, t)
	if err != nil {
		return nil, fmt.Errorf("parsing timestamp: %w", err)
	}

	return &Metadata{
		Hash:      h,
		Timestamp: ts,
	}, nil

}

// Metadata returns metadata for the database in the archive
func (i *imgArchive) Metadata() (*Metadata, error) {
	return i.metadata, nil
}

type dbArchive struct {
	*os.File
	metadata *Metadata
}

func newDBArchive(dbDir string) (Archive, error) {
	metadata, err := metadataFromFile(filepath.Join(dbDir, "metadata.json"))
	if err != nil {
		return nil, fmt.Errorf("reading metadata: %w", err)
	}

	f, err := os.Open(filepath.Join(dbDir, "tally.db"))
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	return &dbArchive{
		File:     f,
		metadata: metadata,
	}, nil
}

// Metadata returns metadata for the database in the archive
func (i *dbArchive) Metadata() (*Metadata, error) {
	return i.metadata, nil
}
