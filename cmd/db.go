package cmd

import (
	"context"
	"fmt"
	"os"

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
			return err
		}

		dbDir, err := local.Dir()
		if err != nil {
			return fmt.Errorf("getting database path: %w", err)
		}

		mgr, err := local.NewManager(dbDir)
		if err != nil {
			return fmt.Errorf("creating manager: %w", err)
		}

		srcs := []db.Source{
			bqsrc.NewPackageSource(bq),
			bqsrc.NewScoreSource(bq),
			bqsrc.NewCheckSource(bq),
		}
		if err := mgr.CreateDB(ctx, srcs...); err != nil {
			return fmt.Errorf("creating database: %w", err)
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

		dbDir, err := local.Dir()
		if err != nil {
			return fmt.Errorf("getting database path: %w", err)
		}

		mgr, err := local.NewManager(dbDir)
		if err != nil {
			return fmt.Errorf("creating database manager: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Exporting database...\n")
		a, err := mgr.ExportDB()
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

		digest, err := local.WriteArchiveToRemote(ref, a, rOpts...)
		if err != nil {
			return fmt.Errorf("error writing to remote registry: %w", err)
		}
		if bar != nil {
			bar.Close()
		}
		fmt.Fprintf(os.Stdout, "%s", digest)

		return nil
	},
}

var dbPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull the database from a remote registry.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		ref, err := name.ParseReference(args[0])
		if err != nil {
			return fmt.Errorf("parsing reference: %w", err)
		}

		dbDir, err := local.Dir()
		if err != nil {
			return fmt.Errorf("getting database path: %w", err)
		}

		mgr, err := local.NewManager(dbDir)
		if err != nil {
			return fmt.Errorf("creating database manager: %w", err)
		}

		rOpts := []remote.Option{
			remote.WithContext(ctx),
			remote.WithAuthFromKeychain(authn.DefaultKeychain),
		}
		a, err := local.GetArchiveFromRemote(ref, rOpts...)
		if err != nil {
			return fmt.Errorf("getting archive from remote image: %w", err)
		}

		update, err := mgr.NeedsUpdate(a)
		if err != nil {
			return fmt.Errorf("checking update status: %w", err)
		}
		if !update {
			fmt.Fprintf(os.Stderr, "Database already up-to-date.\n")
			return nil
		}

		fmt.Fprintf(os.Stderr, "Pulling database from %q...\n", args[0])
		if err := mgr.ImportDB(a); err != nil {
			return fmt.Errorf("importing database: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Database pulled successfully\n")

		return nil
	},
}
