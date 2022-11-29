package manager

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/static"
	"github.com/klauspost/compress/zstd"
)

func TestManagerPullDB(t *testing.T) {
	testCases := map[string]struct {
		setup        func(t *testing.T, dir string) v1.Image
		wantMetadata *Metadata
		wantData     []byte
		wantErr      bool
		wantUpdated  bool
	}{
		// The simple case: import the database for the first time
		"import": {
			setup: func(t *testing.T, dir string) v1.Image {
				data := []byte("foobar")

				metadata, err := json.Marshal(&Metadata{
					SHA256: fmt.Sprintf("%x", sha256.Sum256(data)),
				})
				if err != nil {
					t.Fatalf("unexpected error marshaling metadata to json: %s", err)
				}

				dbLayer, err := newDBLayer(data)
				if err != nil {
					t.Fatalf("unexpected error creating db layer: %s", err)
				}

				img, err := mutate.AppendLayers(
					empty.Image,
					static.NewLayer(metadata, metadataMediaType),
					dbLayer,
				)
				if err != nil {
					t.Fatalf("unexpected error appending layers: %s", err)
				}

				return img
			},
			wantMetadata: &Metadata{
				SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte("foobar"))),
			},
			wantData:    []byte("foobar"),
			wantUpdated: true,
		},
		// Import the existing database
		"no update required": {
			setup: func(t *testing.T, dir string) v1.Image {
				data := []byte("foobar")
				if err := os.WriteFile(filepath.Join(dir, "tally.db"), data, os.ModePerm); err != nil {
					t.Fatalf("unexpected error writing database file: %s", err)
				}
				h := sha256.Sum256(data)
				meta := &Metadata{
					SHA256: fmt.Sprintf("%x", h),
				}
				f, err := os.Create(filepath.Join(dir, "metadata.json"))
				if err != nil {
					t.Fatalf("unexpected error creating metadata file: %s", err)
				}
				defer f.Close()
				if err := json.NewEncoder(f).Encode(meta); err != nil {
					t.Fatalf("unexpected error writing to metadata file: %s", err)
				}

				metadata, err := json.Marshal(meta)
				if err != nil {
					t.Fatalf("unexpected error marshaling metadata to json: %s", err)
				}

				dbLayer, err := newDBLayer(data)
				if err != nil {
					t.Fatalf("unexpected error creating db layer: %s", err)
				}

				img, err := mutate.AppendLayers(
					empty.Image,
					static.NewLayer(metadata, metadataMediaType),
					dbLayer,
				)
				if err != nil {
					t.Fatalf("unexpected error appending layers: %s", err)
				}

				return img
			},
			wantMetadata: &Metadata{
				SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte("foobar"))),
			},
			wantData: []byte("foobar"),
		},
		// Import the database and overwrite an existing one
		"overwrite": {
			setup: func(t *testing.T, dir string) v1.Image {
				oldData := []byte("barfoo")
				if err := os.WriteFile(filepath.Join(dir, "tally.db"), oldData, os.ModePerm); err != nil {
					t.Fatalf("unexpected error writing database file: %s", err)
				}
				oldHash := sha256.Sum256(oldData)
				oldMeta := &Metadata{
					SHA256: fmt.Sprintf("%x", oldHash),
				}
				f, err := os.Create(filepath.Join(dir, "metadata.json"))
				if err != nil {
					t.Fatalf("unexpected error creating metadata file: %s", err)
				}
				defer f.Close()
				if err := json.NewEncoder(f).Encode(oldMeta); err != nil {
					t.Fatalf("unexpected error writing to metadata file: %s", err)
				}

				// Return an image that should overwrite the
				// existing database
				data := []byte("foobar")

				metadata, err := json.Marshal(&Metadata{
					SHA256: fmt.Sprintf("%x", sha256.Sum256(data)),
				})
				if err != nil {
					t.Fatalf("unexpected error marshaling metadata to json: %s", err)
				}

				dbLayer, err := newDBLayer(data)
				if err != nil {
					t.Fatalf("unexpected error creating db layer: %s", err)
				}

				img, err := mutate.AppendLayers(
					empty.Image,
					static.NewLayer(metadata, metadataMediaType),
					dbLayer,
				)
				if err != nil {
					t.Fatalf("unexpected error appending layers: %s", err)
				}

				return img
			},
			wantMetadata: &Metadata{
				SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte("foobar"))),
			},
			wantData:    []byte("foobar"),
			wantUpdated: true,
		},
		"metadata error": {
			setup: func(t *testing.T, dir string) v1.Image {
				data := []byte("foobar")

				dbLayer, err := newDBLayer(data)
				if err != nil {
					t.Fatalf("unexpected error creating db layer: %s", err)
				}

				img, err := mutate.AppendLayers(
					empty.Image,
					dbLayer,
				)
				if err != nil {
					t.Fatalf("unexpected error appending layers: %s", err)
				}

				return img
			},
			wantErr: true,
		},
		"db error": {
			setup: func(t *testing.T, dir string) v1.Image {
				data := []byte("foobar")

				metadata, err := json.Marshal(&Metadata{
					SHA256: fmt.Sprintf("%x", sha256.Sum256(data)),
				})
				if err != nil {
					t.Fatalf("unexpected error marshaling metadata to json: %s", err)
				}

				img, err := mutate.AppendLayers(
					empty.Image,
					static.NewLayer(metadata, metadataMediaType),
				)
				if err != nil {
					t.Fatalf("unexpected error appending layers: %s", err)
				}

				return img
			},
			wantErr: true,
		},
		"random image error": {
			setup: func(t *testing.T, dir string) v1.Image {
				img, err := random.Image(1024, 2)
				if err != nil {
					t.Fatalf("unexpected error creating image: %s", err)
				}

				return img
			},
			wantErr: true,
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			tmpDir, err := ioutil.TempDir("", "tally-test")
			if err != nil {
				t.Fatalf("unexpected error creating temporary directory: %s", err)
			}
			t.Cleanup(func() { os.RemoveAll(tmpDir) })

			img := tc.setup(t, tmpDir)

			host := setupRegistry(t)

			tag := fmt.Sprintf("%s/%s:%s", host, "repo", "tag")
			ref, err := name.ParseReference(tag)
			if err != nil {
				t.Fatalf("unexpected error parsing reference: %s", err)
			}
			if err := remote.Write(ref, img); err != nil {
				t.Fatalf("unexpected error writing image to registry: %s", err)
			}

			mgr, err := NewManager(WithDir(tmpDir))
			if err != nil {
				t.Fatalf("unexpected error creating manager: %s", err)
			}

			gotUpdated, err := mgr.PullDB(context.Background(), tag)
			if err != nil && !tc.wantErr {
				t.Fatalf("unexpected error pulling db: %s", err)
			}
			if err == nil && tc.wantErr {
				t.Fatalf("expected error pulling db but got nil")
			}

			if !tc.wantErr {
				if gotUpdated != tc.wantUpdated {
					t.Fatalf("expected updated value %t but got %t", tc.wantUpdated, gotUpdated)
				}

				m, err := mgr.Metadata()
				if err != nil {
					t.Fatalf("unexpected error getting manager metadata: %s", err)
				}

				if diff := cmp.Diff(tc.wantMetadata, m); diff != "" {
					t.Fatalf("unexpected metadata:\n%s", diff)
				}

				gotData, err := ioutil.ReadFile(filepath.Join(tmpDir, "tally.db"))
				if err != nil {
					t.Fatalf("unexpected error reading file: %s", err)
				}

				if !cmp.Equal(tc.wantData, gotData) {
					t.Fatalf("unexpected extracted data, wanted %q but got %q", tc.wantData, gotData)
				}
			}
		})
	}
}

func newDBLayer(data []byte) (v1.Layer, error) {
	var buf bytes.Buffer
	enc, err := zstd.NewWriter(&buf)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(enc, bytes.NewReader(data)); err != nil {
		enc.Close()
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}

	return static.NewLayer(buf.Bytes(), dbMediaType), nil
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
