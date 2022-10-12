package oci

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/jetstack/tally/internal/db/local"
)

func TestGetArchive(t *testing.T) {
	testCases := map[string]func(t *testing.T) (v1.Image, bool){
		"happy path": func(t *testing.T) (v1.Image, bool) {
			img, err := mutate.AppendLayers(
				empty.Image,
				static.NewLayer([]byte(`foobar`), metadataMediaType),
				static.NewLayer([]byte(`barfoo`), dbMediaType),
			)
			if err != nil {
				t.Fatalf("unexpected error creating image: %s", err)
			}

			return img, false
		},
		"should return an error when the reference isn't in the registry": func(t *testing.T) (v1.Image, bool) {
			return nil, true
		},
		"should return an error when the metadata layer is missing": func(t *testing.T) (v1.Image, bool) {
			img, err := mutate.AppendLayers(
				empty.Image,
				static.NewLayer([]byte(`barfoo`), dbMediaType),
			)
			if err != nil {
				t.Fatalf("unexpected error creating image: %s", err)
			}

			return img, true
		},
		"should return an error when the db layer is missing": func(t *testing.T) (v1.Image, bool) {
			img, err := mutate.AppendLayers(
				empty.Image,
				static.NewLayer([]byte(`foobar`), metadataMediaType),
			)
			if err != nil {
				t.Fatalf("unexpected error creating image: %s", err)
			}

			return img, true
		},
		"should return an error when there is an extra layer": func(t *testing.T) (v1.Image, bool) {
			img, err := mutate.AppendLayers(
				empty.Image,
				static.NewLayer([]byte(`foobar`), metadataMediaType),
				static.NewLayer([]byte(`barfoo`), dbMediaType),
				static.NewLayer([]byte(`bazfoo`), types.DockerLayer),
			)
			if err != nil {
				t.Fatalf("unexpected error creating image: %s", err)
			}

			return img, true
		},
		"should return an error when there are no layers": func(t *testing.T) (v1.Image, bool) {
			return empty.Image, true
		},
	}

	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			// Create a test registry
			host := setupRegistry(t)

			img, wantErr := tc(t)

			// Upload image to a reference in the test registry
			tag := fmt.Sprintf("%s/%s:%s", host, "repo", "tag")
			ref, err := name.ParseReference(tag)
			if err != nil {
				t.Fatalf("unexpected error parsing image reference: %s", err)
			}
			if img != nil {
				if err := remote.Write(ref, img); err != nil {
					t.Fatalf("unexpected error writing test image to registry: %s", err)
				}
			}

			// Run GetArchive
			_, err = GetArchive(tag)
			if err != nil && !wantErr {
				t.Fatalf("unexpected error running GetArchive: %s", err)
			}
			if err == nil && wantErr {
				t.Fatalf("expected error but got nil")
			}
		})
	}
}

func TestArchiveDB(t *testing.T) {
	testCases := map[string]func(t *testing.T) (*Archive, []byte, bool){
		"happy path": func(t *testing.T) (*Archive, []byte, bool) {
			data := []byte(`foobar`)

			var buf bytes.Buffer
			gw := gzip.NewWriter(&buf)
			if _, err := gw.Write(data); err != nil {
				t.Fatalf("unexpected error writing gzip: %s", err)
			}
			if err := gw.Close(); err != nil {
				t.Fatalf("unexpected error closing gzip writer: %s", err)
			}
			img, err := mutate.AppendLayers(
				empty.Image,
				static.NewLayer(buf.Bytes(), dbMediaType),
			)
			if err != nil {
				t.Fatalf("unexpected error creating image: %s", err)
			}

			return &Archive{
				img: img,
				opts: &options{
					pbWriter: io.Discard,
				},
			}, data, false
		},
		"should return error when layer is missing": func(t *testing.T) (*Archive, []byte, bool) {
			return &Archive{
				img: empty.Image,
				opts: &options{
					pbWriter: io.Discard,
				},
			}, []byte{}, true
		},
		"should return an error when layer is not compressed": func(t *testing.T) (*Archive, []byte, bool) {
			img, err := mutate.AppendLayers(
				empty.Image,
				static.NewLayer([]byte(`foobar`), dbMediaType),
			)
			if err != nil {
				t.Fatalf("unexpected error creating image: %s", err)
			}

			return &Archive{
				img: img,
				opts: &options{
					pbWriter: io.Discard,
				},
			}, []byte{}, true
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			a, wantData, wantErr := tc(t)

			gotData, err := a.DB()
			if err != nil && !wantErr {
				t.Fatalf("unexpected error: %s", err)
			}
			if err == nil && wantErr {
				t.Fatalf("wanted error but got nil")
			}

			if wantErr {
				return
			}

			if diff := cmp.Diff(wantData, gotData); diff != "" {
				t.Fatalf("unexpected data:\n%s", diff)
			}
		})
	}
}

func TestArchiveMetadata(t *testing.T) {
	testCases := map[string]func(t *testing.T) (*Archive, *local.Metadata, bool){
		"happy path": func(t *testing.T) (*Archive, *local.Metadata, bool) {
			metadata := &local.Metadata{
				SHA256: "foobar",
			}
			data, err := json.Marshal(metadata)
			if err != nil {
				t.Fatalf("unexpected error encoding json: %s", err)
			}

			img, err := mutate.AppendLayers(
				empty.Image,
				static.NewLayer(data, metadataMediaType),
			)
			if err != nil {
				t.Fatalf("unexpected error creating image: %s", err)
			}

			return &Archive{
				img: img,
				opts: &options{
					pbWriter: io.Discard,
				},
			}, metadata, false
		},
		"should return error when layer is missing": func(t *testing.T) (*Archive, *local.Metadata, bool) {
			return &Archive{
				img: empty.Image,
				opts: &options{
					pbWriter: io.Discard,
				},
			}, nil, true
		},
		"should return an error when layer is not json": func(t *testing.T) (*Archive, *local.Metadata, bool) {
			img, err := mutate.AppendLayers(
				empty.Image,
				static.NewLayer([]byte(`foobar`), metadataMediaType),
			)
			if err != nil {
				t.Fatalf("unexpected error creating image: %s", err)
			}

			return &Archive{
				img: img,
				opts: &options{
					pbWriter: io.Discard,
				},
			}, nil, true
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			a, wantMeta, wantErr := tc(t)

			gotMeta, err := a.Metadata()
			if err != nil && !wantErr {
				t.Fatalf("unexpected error: %s", err)
			}
			if err == nil && wantErr {
				t.Fatalf("wanted error but got nil")
			}

			if wantErr {
				return
			}

			if diff := cmp.Diff(wantMeta, gotMeta); diff != "" {
				t.Fatalf("unexpected data:\n%s", diff)
			}
		})
	}
}

func setupRegistry(t *testing.T) string {
	r := httptest.NewServer(registry.New())
	t.Cleanup(r.Close)
	u, err := url.Parse(r.URL)
	if err != nil {
		t.Fatalf("unexpected error parsing registry url: %s", err)
	}
	return u.Host
}
