package local

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
)

func TestArchiveWriteGetFromRemote(t *testing.T) {
	// Setup a test registry
	r := httptest.NewServer(registry.New())
	u, err := url.Parse(r.URL)
	if err != nil {
		t.Fatalf("unexpected error parsing registry url: %s", err)
	}
	defer r.Close()

	// Create a reference in the registry
	ref, err := name.ParseReference(fmt.Sprintf("%s/%s:%s", u.Host, "repo", "tag"))
	if err != nil {
		t.Fatalf("unexpected error parsing reference: %s", err)
	}

	wantDB := []byte("foobar")

	h := sha256.New()
	if _, err := io.Copy(h, bytes.NewReader(wantDB)); err != nil {
		t.Fatalf("unexpected error writing hash: %s", err)
	}

	wantMetadata := &Metadata{
		Hash:      fmt.Sprintf("%x", h.Sum(nil)),
		Timestamp: time.Now(),
	}

	// Create an archive
	wantArchive := &mockArchive{
		ReadCloser: ioutil.NopCloser(bytes.NewReader(wantDB)),
		metadata: func() (*Metadata, error) {
			return wantMetadata, nil
		},
	}

	// Write the archive to the registry
	if err := WriteArchiveToRemote(ref, wantArchive); err != nil {
		t.Fatalf("unexpected error writing archive to registry: %s", err)
	}

	// Get the archive from the registry
	gotArchive, err := GetArchiveFromRemote(ref)
	if err != nil {
		t.Fatalf("unepected error getting archive from registry: %s", err)
	}

	// Test that the archive we get is the one we uploaded
	gotMetadata, err := gotArchive.Metadata()
	if err != nil {
		t.Fatalf("unexpected error getting metadata from archive: %s", err)
	}

	if diff := cmp.Diff(wantMetadata, gotMetadata); diff != "" {
		t.Fatalf("unexpected metadata:\n%s", diff)
	}

	gotDB, err := ioutil.ReadAll(gotArchive)
	if err != nil {
		t.Fatalf("unexpected error reading from archive: %s", err)
	}
	if diff := cmp.Diff(wantDB, gotDB); diff != "" {
		t.Fatalf("unexpected database bytes:\n%s", diff)
	}
}
