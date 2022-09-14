package local

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/db"
)

func TestExportImportDB(t *testing.T) {
	exportDir, err := ioutil.TempDir("", "tally-db-export")
	if err != nil {
		t.Fatalf("unexpected error creating temp dir: %s", err)
	}
	defer os.RemoveAll(exportDir)

	importDir, err := ioutil.TempDir("", "tally-db-import")
	if err != nil {
		t.Fatalf("unexpected error creating temp dir: %s", err)
	}
	defer os.RemoveAll(importDir)

	exportMgr := &manager{
		dbDir: exportDir,
	}

	// Test that we get an error when the database doesn't exist
	if _, err := exportMgr.ExportDB(); err == nil {
		t.Fatalf("expected error exporting non-existent database but got nil")
	}

	// Create the database
	if err := exportMgr.CreateDB(context.Background()); err != nil {
		t.Fatalf("unexpected error creating database: %s", err)
	}

	// Export the database
	a, err := exportMgr.ExportDB()
	if err != nil {
		t.Fatalf("unexpected error exporting database: %s", err)
	}

	// Import the database to another path
	importMgr := &manager{
		dbDir: importDir,
	}
	if err := importMgr.ImportDB(a); err != nil {
		t.Fatalf("unexpected error importing database: %s", err)
	}

	// Ensure that the metadata of the imported database matches the
	// exported one
	importMetadata, err := importMgr.Metadata()
	if err != nil {
		t.Fatalf("unexpected error getting metadata: %s", err)
	}
	exportMetadata, err := exportMgr.Metadata()
	if err != nil {
		t.Fatalf("unexpected error getting metadata: %s", err)
	}
	if diff := cmp.Diff(exportMetadata, importMetadata); diff != "" {
		t.Fatalf("unxpected metadata:\n%s", diff)
	}
}

type mockSrc struct {
	name   string
	update func(context.Context, db.DBWriter) error
}

func (m *mockSrc) String() string {
	return m.name
}
func (m *mockSrc) Update(ctx context.Context, tallyDB db.DBWriter) error {
	if m.update == nil {
		return nil
	}

	return m.update(ctx, tallyDB)
}

func TestManagerCreateDB(t *testing.T) {
	dbDir, err := ioutil.TempDir("", "tally-db")
	if err != nil {
		t.Fatalf("unexpected error creating temp dir: %s", err)
	}
	defer os.RemoveAll(dbDir)

	mgr, err := NewManager(dbDir)
	if err != nil {
		t.Fatalf("unexpected error creating manager: %s", err)
	}

	wantPkgs := map[db.Package][]string{
		{
			System: "GO",
			Name:   "foobar",
		}: {
			"github.com/baz/foo",
			"github.com/foo/bar",
		},
		{
			System: "NPM",
			Name:   "barfoo",
		}: {"github.com/bar/foo"},
	}

	wantScores := map[string]float64{
		"github.com/foo/bar": 8.4,
		"github.com/bar/foo": 4.2,
	}

	wantChecks := map[string][]db.Check{
		"github.com/foo/bar": {
			{
				Name:       "bar",
				Score:      5,
				Repository: "github.com/foo/bar",
			},
			{
				Name:       "foo",
				Score:      8,
				Repository: "github.com/foo/bar",
			},
		},
		"github.com/bar/foo": {
			{
				Name:       "bar",
				Score:      2,
				Repository: "github.com/bar/foo",
			},
			{
				Name:       "foo",
				Score:      7,
				Repository: "github.com/bar/foo",
			},
		},
	}

	src := &mockSrc{
		name: "mock",
		update: func(ctx context.Context, tallyDB db.DBWriter) error {
			var pkgs []db.Package
			for pkg, repos := range wantPkgs {
				for _, repo := range repos {
					pkgs = append(pkgs, db.Package{
						System:     pkg.System,
						Name:       pkg.Name,
						Repository: repo,
					})
				}
			}
			if err := tallyDB.AddPackages(ctx, pkgs...); err != nil {
				t.Fatalf("unexpected error adding packages: %s", err)
			}

			var scores []db.Score
			for repo, score := range wantScores {
				scores = append(scores, db.Score{
					Repository: repo,
					Score:      score,
				})
			}
			if err := tallyDB.AddScores(ctx, scores...); err != nil {
				t.Fatalf("unexpected error adding scores: %s", err)
			}

			for _, checks := range wantChecks {
				if err := tallyDB.AddChecks(ctx, checks...); err != nil {
					t.Fatalf("unexpected error adding checks: %s", err)
				}
			}

			return nil
		},
	}

	if err := mgr.CreateDB(context.Background(), src); err != nil {
		t.Fatalf("unexpected error creating database with mock src: %s", err)
	}

	tallyDB, err := mgr.DB()
	if err != nil {
		t.Fatalf("unexpected error getting db: %s", err)
	}
	defer tallyDB.Close()

	for pkg, wantRepos := range wantPkgs {
		gotRepos, err := tallyDB.GetRepositories(context.Background(), pkg.System, pkg.Name)
		if err != nil {
			t.Fatalf("unexpected error getting repositories: %s", err)
		}
		if diff := cmp.Diff(wantRepos, gotRepos); diff != "" {
			t.Fatalf("unexpected repositories:\n%s", diff)
		}
	}

	for repo, wantScore := range wantScores {
		gotScores, err := tallyDB.GetScores(context.Background(), repo)
		if err != nil {
			t.Fatalf("unexpected error getting score: %s", err)
		}
		wantScores := []db.Score{
			{
				Repository: repo,
				Score:      wantScore,
			},
		}
		if diff := cmp.Diff(wantScores, gotScores); diff != "" {
			t.Fatalf("unexpected scores:\n%s", diff)
		}
	}

	for repo, checks := range wantChecks {
		gotChecks, err := tallyDB.GetChecks(context.Background(), repo)
		if err != nil {
			t.Fatalf("unexpected error getting checks: %s", err)
		}
		if diff := cmp.Diff(checks, gotChecks); diff != "" {
			t.Fatalf("unexpected checks for %s:\n%s", repo, diff)
		}
	}

	metadata, err := mgr.Metadata()
	if err != nil {
		t.Fatalf("unexpected error getting metadata: %s", err)
	}
	if metadata == nil {
		t.Fatalf("unexpected nil metadata")
	}
}

func TestManagerCreateDB_UpdateError(t *testing.T) {
	dbDir, err := ioutil.TempDir("", "tally-db")
	if err != nil {
		t.Fatalf("unexpected error creating temp dir: %s", err)
	}
	defer os.RemoveAll(dbDir)

	mgr, err := NewManager(dbDir)
	if err != nil {
		t.Fatalf("unexpected error creating manager: %s", err)
	}

	srcs := []db.Source{
		&mockSrc{
			name: "adds packages",
			update: func(ctx context.Context, tallyDB db.DBWriter) error {
				if err := tallyDB.AddPackages(ctx, []db.Package{
					{
						System:     "GO",
						Name:       "foobar",
						Repository: "github.com/foo/bar",
					},
					{
						System:     "NPM",
						Name:       "barfoo",
						Repository: "github.com/bar/foo",
					},
				}...); err != nil {
					t.Fatalf("unexpected error adding packages: %s", err)
				}

				return nil
			},
		},
		&mockSrc{
			name: "returns an error",
			update: func(ctx context.Context, tallyDB db.DBWriter) error {
				return fmt.Errorf("gerror")
			},
		},
		&mockSrc{
			name: "shouldn't be called",
			update: func(ctx context.Context, tallyDB db.DBWriter) error {
				t.Fatalf("unexpected call to third source in slice")
				return nil
			},
		},
	}

	if err := mgr.CreateDB(context.Background(), srcs...); err == nil {
		t.Fatalf("expected error creating database but got nil")
	}

	// The create operation should have aborted, so the metadata should be
	// nil
	metadata, err := mgr.Metadata()
	if err != nil {
		t.Fatalf("unexpected error getting database metadata: %s", err)
	}
	if metadata != nil {
		t.Fatalf("expected metadata to be nil but got: %+v", metadata)
	}
}

func TestManagerCreateDB_Overwrite(t *testing.T) {
	dbDir, err := ioutil.TempDir("", "tally-db")
	if err != nil {
		t.Fatalf("unexpected error creating temp dir: %s", err)
	}
	defer os.RemoveAll(dbDir)

	mgr, err := NewManager(dbDir)
	if err != nil {
		t.Fatalf("unexpected error creating manager: %s", err)
	}
	if err := mgr.CreateDB(context.Background()); err != nil {
		t.Fatalf("unexpected error creating database: %s", err)
	}

	oldMeta, err := mgr.Metadata()
	if err != nil {
		t.Fatalf("unexpected error getting metadata: %s", err)
	}

	src := &mockSrc{
		name: "mock",
		update: func(ctx context.Context, tallyDB db.DBWriter) error {
			if err := tallyDB.AddPackages(ctx, []db.Package{
				{
					System:     "GO",
					Name:       "foobar",
					Repository: "github.com/foo/bar",
				},
				{
					System:     "NPM",
					Name:       "barfoo",
					Repository: "github.com/bar/foo",
				},
			}...); err != nil {
				t.Fatalf("unexpected error adding packages: %s", err)
			}

			return nil
		},
	}

	if err := mgr.CreateDB(context.Background(), src); err != nil {
		t.Fatalf("unexpected error creating database with mock src: %s", err)
	}

	// The metadata should have changed
	newMeta, err := mgr.Metadata()
	if err != nil {
		t.Fatalf("unexpected error getting new metadata: %s", err)
	}
	if newMeta.Equal(*oldMeta) {
		t.Fatalf("expected metadata of overwritten database to be different, but it's the same")
	}
}

type mockArchive struct {
	io.ReadCloser
	metadata func() (*Metadata, error)
}

func (m *mockArchive) Metadata() (*Metadata, error) {
	if m.metadata == nil {
		return nil, nil
	}

	return m.metadata()
}

func TestManagerNeedsUpdate(t *testing.T) {
	testCases := map[string]struct {
		newArchive  func(t *testing.T) Archive
		metadata    []byte
		needsUpdate bool
		wantErr     bool
	}{
		"needs update": {
			newArchive: func(t *testing.T) Archive {
				return &mockArchive{
					ReadCloser: ioutil.NopCloser(bytes.NewBuffer([]byte("foobar"))),
					metadata: func() (*Metadata, error) {
						return &Metadata{
							Hash: "foobar",
						}, nil
					},
				}
			},
			metadata:    []byte(`{"hash":"notfoobar"}`),
			needsUpdate: true,
		},
		"needs update nil": {
			newArchive: func(t *testing.T) Archive {
				return &mockArchive{
					ReadCloser: ioutil.NopCloser(bytes.NewBuffer([]byte("foobar"))),
					metadata: func() (*Metadata, error) {
						return &Metadata{
							Hash: "foobar",
						}, nil
					},
				}
			},
			needsUpdate: true,
		},
		"doesn't need update": {
			newArchive: func(t *testing.T) Archive {
				return &mockArchive{
					ReadCloser: ioutil.NopCloser(bytes.NewBuffer([]byte("foobar"))),
					metadata: func() (*Metadata, error) {
						return &Metadata{
							Hash: "foobar",
						}, nil
					},
				}
			},
			metadata: []byte(`{"hash":"foobar"}`),
		},
		"archive error": {
			newArchive: func(t *testing.T) Archive {
				return &mockArchive{
					ReadCloser: ioutil.NopCloser(bytes.NewBuffer([]byte("foobar"))),
					metadata: func() (*Metadata, error) {
						return nil, errors.New("error")
					},
				}
			},
			wantErr: true,
		},
		"metadata error": {
			newArchive: func(t *testing.T) Archive {
				return &mockArchive{
					ReadCloser: ioutil.NopCloser(bytes.NewBuffer([]byte("foobar"))),
					metadata: func() (*Metadata, error) {
						return &Metadata{
							Hash: "foobar",
						}, nil
					},
				}
			},
			metadata: []byte(`notjson`),
			wantErr:  true,
		},
	}

	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			tmpDir, err := ioutil.TempDir("", "tally-test")
			if err != nil {
				t.Fatalf("unexpected error creating temporary directory: %s", err)
			}

			if len(tc.metadata) > 0 {
				if err := os.WriteFile(filepath.Join(tmpDir, "metadata.json"), tc.metadata, os.ModePerm); err != nil {
					t.Fatalf("unexpected error creating metadata file: %s", err)
				}
			}

			mgr := &manager{tmpDir}

			needsUpdate, err := mgr.NeedsUpdate(tc.newArchive(t))
			if err != nil && !tc.wantErr {
				t.Fatalf("unexpected error running NeedsUpdate: %s", err)
			}
			if err == nil && tc.wantErr {
				t.Fatalf("expected error but got nil")
			}

			if needsUpdate != tc.needsUpdate {
				t.Fatalf("expected %t but got %t", tc.needsUpdate, needsUpdate)
			}
		})
	}
}
