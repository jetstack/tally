package local

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jetstack/tally/internal/db"
)

type mockArchive struct {
	db       func() ([]byte, error)
	metadata func() (*Metadata, error)
}

func (m *mockArchive) DB() ([]byte, error) {
	if m.db == nil {
		return []byte{}, nil
	}

	return m.db()
}

func (m *mockArchive) Metadata() (*Metadata, error) {
	if m.metadata == nil {
		return nil, nil
	}

	return m.metadata()
}

var testErr = errors.New("error")

func TestManagerUpdateDB(t *testing.T) {
	testCases := map[string]struct {
		setup       func(t *testing.T, dir string) Archive
		wantData    []byte
		wantErr     error
		wantUpdated bool
	}{
		// The simple case: import the database for the first time
		"import": {
			setup: func(t *testing.T, dir string) Archive {
				data := []byte("foobar")
				h := sha256.Sum256(data)
				return &mockArchive{
					db: func() ([]byte, error) {
						return data, nil
					},
					metadata: func() (*Metadata, error) {
						return &Metadata{
							SHA256: fmt.Sprintf("%x", h),
						}, nil
					},
				}
			},
			wantData:    []byte("foobar"),
			wantUpdated: true,
		},
		// Import the existing database
		"no update required": {
			setup: func(t *testing.T, dir string) Archive {
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

				return &mockArchive{
					db: func() ([]byte, error) {
						return data, nil
					},
					metadata: func() (*Metadata, error) {
						return meta, nil
					},
				}
			},
			wantData: []byte("foobar"),
		},
		// Import the database and overwrite an existing one
		"overwrite": {
			setup: func(t *testing.T, dir string) Archive {
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

				// Return an archive that should overwrite the
				// existing database
				data := []byte("foobar")
				h := sha256.Sum256(data)
				return &mockArchive{
					db: func() ([]byte, error) {
						return data, nil
					},
					metadata: func() (*Metadata, error) {
						return &Metadata{
							SHA256: fmt.Sprintf("%x", h),
						}, nil
					},
				}
			},
			wantData:    []byte("foobar"),
			wantUpdated: true,
		},
		"metadata error": {
			setup: func(t *testing.T, dir string) Archive {
				return &mockArchive{
					db: func() ([]byte, error) {
						return nil, fmt.Errorf("unexpected call to DB")
					},
					metadata: func() (*Metadata, error) {
						return nil, testErr
					},
				}
			},
			wantErr: testErr,
		},
		"db error": {
			setup: func(t *testing.T, dir string) Archive {
				h := sha256.Sum256([]byte("foobar"))
				return &mockArchive{
					db: func() ([]byte, error) {
						return nil, testErr
					},
					metadata: func() (*Metadata, error) {
						return &Metadata{
							SHA256: fmt.Sprintf("%x", h),
						}, nil
					},
				}
			},
			wantErr: testErr,
		},
	}
	for n, tc := range testCases {
		t.Run(n, func(t *testing.T) {
			tmpDir, err := ioutil.TempDir("", "tally-test")
			if err != nil {
				t.Fatalf("unexpected error creating temporary directory: %s", err)
			}

			a := tc.setup(t, tmpDir)

			mgr, err := NewManager(WithDir(tmpDir))
			if err != nil {
				t.Fatalf("unexpected error creating manager: %s", err)
			}

			gotUpdated, err := mgr.UpdateDB(a)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error %v but got %v", tc.wantErr, errors.Unwrap(err))
			}

			if tc.wantErr == nil {
				if gotUpdated != tc.wantUpdated {
					t.Fatalf("expected updated value %t but got %t", tc.wantUpdated, gotUpdated)
				}
				am, err := a.Metadata()
				if err != nil {
					t.Fatalf("unexpected error getting archive metadata: %s", err)
				}

				m, err := mgr.Metadata()
				if err != nil {
					t.Fatalf("unexpected error getting manager metadata: %s", err)
				}

				if diff := cmp.Diff(am, m); diff != "" {
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

	mgr, err := NewManager(WithDir(dbDir))
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

	mgr, err := NewManager(WithDir(dbDir))
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

	mgr, err := NewManager(WithDir(dbDir))
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
	if newMeta.Equals(*oldMeta) {
		t.Fatalf("expected metadata of overwritten database to be different, but it's the same")
	}
}
