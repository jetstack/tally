package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"cloud.google.com/go/bigquery"

	"github.com/jetstack/tally/internal/db"
	"github.com/jetstack/tally/internal/db/db/local"
	bqsrc "github.com/jetstack/tally/internal/db/source/bigquery"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(dbCmd)

	dbCmd.AddCommand(dbCreateCmd)
	dbCreateCmd.Flags().StringVarP(&dbco.ProjectID, "project-id", "p", "", "Google Cloud project that Big Query requests are billed against")
	dbCreateCmd.MarkFlagRequired("project-id")
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
