package cmd

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/bigquery"

	"github.com/jetstack/tally/internal/db"
	"github.com/jetstack/tally/internal/depsdev"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(dbCmd)

	dbCmd.AddCommand(dbCreateCmd)
	dbCreateCmd.Flags().StringVarP(&dbco.ProjectID, "project-id", "p", "", "Google Cloud project that Big Query requests are billed against")
	dbCreateCmd.MarkFlagRequired("project-id")

	dbCmd.AddCommand(dbPullCmd)
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
			return fmt.Errorf("creating big query client: %w", err)
		}

		mgr, err := db.NewManager("", os.Stderr)
		if err != nil {
			return fmt.Errorf("creating database manager: %w", err)
		}

		if err := mgr.CreateDB(ctx, depsdev.NewDBSource(bq)); err != nil {
			return fmt.Errorf("creating database: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Database created successfully.\n")

		return nil
	},
}

var dbPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull the database from a remote registry.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		mgr, err := db.NewManager("", os.Stderr)
		if err != nil {
			return fmt.Errorf("creating database manager: %w", err)
		}

		updated, err := mgr.PullDB(ctx, args[0])
		if err != nil {
			return fmt.Errorf("importing database: %w", err)
		}
		if updated {
			fmt.Fprintf(os.Stderr, "Database pulled successfully\n")
		} else {
			fmt.Fprintf(os.Stderr, "Database already up-to-date.\n")
		}

		return nil
	},
}
