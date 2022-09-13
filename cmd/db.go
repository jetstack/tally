package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"cloud.google.com/go/bigquery"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/jetstack/tally/internal/db"
	"github.com/jetstack/tally/internal/db/db/local"
	bqsrc "github.com/jetstack/tally/internal/db/source/bigquery"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(dbCmd)

	dbCmd.AddCommand(dbCreateCmd)
	dbCreateCmd.Flags().StringVarP(&dbco.ProjectID, "project-id", "p", "", "Google Cloud project that Big Query requests are billed against")
	dbCreateCmd.MarkFlagRequired("project-id")

	dbCmd.AddCommand(dbPushCmd)
}

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database",
}

type dbCreateOptions struct {
	ProjectID string
}

var dbco dbCreateOptions

var dbCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create the tally database.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		bq, err := bigquery.NewClient(ctx, dbco.ProjectID)
		if err != nil {
			return err
		}

		dbPath, err := local.Path()
		if err != nil {
			return fmt.Errorf("getting database path: %w", err)
		}

		if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
			return fmt.Errorf("database already exists: %s", dbPath)
		}

		if err := os.MkdirAll(filepath.Dir(dbPath), os.ModePerm); err != nil {
			return fmt.Errorf("creating database path: %w", err)
		}

		tallyDB, err := local.NewDB(dbPath)
		if err != nil {
			return fmt.Errorf("creating database client: %w", err)
		}
		defer tallyDB.Close()

		fmt.Fprintf(os.Stderr, "Initializing database...\n")
		if err := tallyDB.Initialize(ctx); err != nil {
			return fmt.Errorf("initializing database: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Database initialized.\n")

		srcs := []db.Source{
			bqsrc.NewPackageSource(bq, tallyDB),
			bqsrc.NewScoreSource(bq, tallyDB),
			bqsrc.NewCheckSource(bq, tallyDB),
		}
		for _, src := range srcs {
			fmt.Fprintf(os.Stderr, "Populating database from source %q...\n", src)
			if err := src.Update(ctx); err != nil {
				return fmt.Errorf("populating database from source: %q: %w", src, err)
			}
		}

		fmt.Fprintf(os.Stderr, "Database created successfully.\n")

		return nil
	},
}

var dbPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push the database to a remote registry.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref, err := name.ParseReference(args[0])
		if err != nil {
			return fmt.Errorf("parsing reference: %w", err)
		}

		dbPath, err := local.Path()
		if err != nil {
			return fmt.Errorf("getting database path: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Exporting database...\n")
		img, err := local.ExportDB(dbPath)
		if err != nil {
			return fmt.Errorf("exporting database: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Exported database successfully.\n")

		var bar *progressbar.ProgressBar
		updatesCh := make(chan v1.Update, 100)
		go func() {
			for {
				select {
				case update := <-updatesCh:
					if bar == nil {
						bar = progressbar.DefaultBytes(update.Total)
					}
					bar.Set64(update.Complete)
				}
			}
		}()

		fmt.Fprintf(os.Stderr, "Pushing database to %q...\n", ref)
		rOpts := []remote.Option{
			remote.WithAuthFromKeychain(authn.DefaultKeychain),
			remote.WithProgress(updatesCh),
		}
		if err := remote.Write(ref, img, rOpts...); err != nil {
			return fmt.Errorf("error pushing image: %w", err)
		}
		if bar != nil {
			bar.Close()
		}
		fmt.Fprintf(os.Stderr, "Database pushed successfully.\n")

		return nil
	},
}
