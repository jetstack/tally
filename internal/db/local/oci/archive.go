package oci

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"

	"github.com/cheggaaa/pb/v3"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/jetstack/tally/internal/db/local"
)

const (
	dbMediaType       = types.MediaType("application/vnd.jetstack.tally.db.layer.v1+gzip")
	metadataMediaType = types.MediaType("application/vnd.jetstack.tally.metadata.layer.v1")
)

// Option is a functional option that configures an OCI archive
type Option func(*options)

type options struct {
	pbWriter   io.Writer
	remoteOpts []remote.Option
}

func makeOptions(opts ...Option) *options {
	o := &options{
		pbWriter: io.Discard,
	}
	for _, opt := range opts {
		opt(o)
	}

	return o
}

// WithRemoteOptions is a functional option that configures remote options
func WithRemoteOptions(rOpts ...remote.Option) Option {
	return func(o *options) {
		o.remoteOpts = rOpts
	}
}

// WithProgressBarWriter is a functional option that configures the writer to
// which progress bars are written
func WithProgressBarWriter(w io.Writer) Option {
	return func(o *options) {
		o.pbWriter = w
	}
}

// Archive is an archived database stored in a remote OCI registry. It
// implements local.Archive.
type Archive struct {
	img  v1.Image
	opts *options
}

// GetArchive returns an Archive from a remote OCI registry
func GetArchive(tag string, opts ...Option) (*Archive, error) {
	o := makeOptions(opts...)

	ref, err := name.ParseReference(tag)
	if err != nil {
		return nil, fmt.Errorf("parsing reference: %w", err)
	}

	img, err := remote.Image(ref, o.remoteOpts...)
	if err != nil {
		return nil, fmt.Errorf("getting remote image: %w", err)
	}

	if err := validateImage(img); err != nil {
		return nil, fmt.Errorf("validating image: %w", err)
	}

	return &Archive{
		img:  img,
		opts: o,
	}, nil
}

func validateImage(img v1.Image) error {
	layers, err := img.Layers()
	if err != nil {
		return fmt.Errorf("getting remote layer: %w", err)
	}
	if len(layers) != 2 {
		return fmt.Errorf("unexpected number of layers, wanted %d but got %d", 2, len(layers))
	}
	seen := map[types.MediaType]bool{
		dbMediaType:       false,
		metadataMediaType: false,
	}
	for _, layer := range layers {
		mt, err := layer.MediaType()
		if err != nil {
			return fmt.Errorf("getting media type from layer: %w", err)
		}
		for want := range seen {
			if mt == want {
				seen[mt] = true
			}
		}
	}
	for mt, found := range seen {
		if !found {
			return fmt.Errorf("couldn't find layer with media type: %s", mt)
		}
	}

	return nil
}

// DB returns the contents of the database from the archive
func (a *Archive) DB() ([]byte, error) {
	layers, err := a.img.Layers()
	if err != nil {
		return nil, fmt.Errorf("getting layers: %w", err)
	}

	for _, layer := range layers {
		mt, err := layer.MediaType()
		if err != nil {
			return nil, fmt.Errorf("getting layer media type: %w", err)
		}
		if mt != dbMediaType {
			continue
		}

		size, err := layer.Size()
		if err != nil {
			return nil, fmt.Errorf("getting layer size: %w", err)
		}

		rc, err := layer.Compressed()
		if err != nil {
			return nil, fmt.Errorf("getting uncompressed layer: %w", err)
		}
		defer rc.Close()

		bar := pb.Full.Start64(size)
		bar.SetWriter(a.opts.pbWriter)
		defer bar.Finish()

		gr, err := gzip.NewReader(bar.NewProxyReader(rc))
		if err != nil {
			return nil, fmt.Errorf("creating gzip reader: %w", err)
		}
		defer gr.Close()

		data, err := io.ReadAll(gr)
		if err != nil {
			return nil, fmt.Errorf("reading database contents")
		}

		return data, nil
	}

	return []byte{}, fmt.Errorf("couldn't find db in archive")
}

// Metadata returns the metadata of the database in the archive
func (a *Archive) Metadata() (*local.Metadata, error) {
	layers, err := a.img.Layers()
	if err != nil {
		return nil, fmt.Errorf("getting layers: %w", err)
	}

	for _, layer := range layers {
		mt, err := layer.MediaType()
		if err != nil {
			return nil, fmt.Errorf("getting layer media type: %w", err)
		}

		if mt != metadataMediaType {
			continue
		}

		rc, err := layer.Compressed()
		if err != nil {
			return nil, fmt.Errorf("getting uncompressed layer: %w", err)
		}
		defer rc.Close()

		metadata := &local.Metadata{}
		if err := json.NewDecoder(rc).Decode(metadata); err != nil {
			return nil, fmt.Errorf("decoding metadata json: %w", err)
		}

		return metadata, nil
	}

	return nil, fmt.Errorf("can't find metadata layer in layers")
}
