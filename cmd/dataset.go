package cmd

import (
	"context"
	"fmt"
	"os"

	"cloud.google.com/go/bigquery"
	"github.com/ribbybibby/tally/internal/tally"
	"github.com/spf13/cobra"
)

var datasetCmd = &cobra.Command{
	Use:   "dataset",
	Short: "Dataset related operations.",
	Long:  "Dataset related operations.",
}

type datasetCreateOptions struct {
	ProjectID string
}

var dco datasetCreateOptions

var datasetCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a tally dataset.",
	Long:  "Create a tally dataset.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		bq, err := bigquery.NewClient(ctx, dco.ProjectID)
		if err != nil {
			return fmt.Errorf("creating bigquery client: %w", err)
		}

		dataset, err := tally.NewDataset(bq, args[0])
		if err != nil {
			return fmt.Errorf("creating new dataset: %w", err)
		}

		if err := dataset.Create(ctx); err != nil {
			return fmt.Errorf("creating dataset: %w", err)
		}

		fmt.Fprintf(os.Stderr, "Created dataset: %s\n", dataset)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(datasetCmd)
	datasetCmd.AddCommand(datasetCreateCmd)

	datasetCreateCmd.Flags().StringVarP(&dco.ProjectID, "project-id", "p", "", "Google Cloud project that Big Query requests are billed against")
	datasetCmd.MarkFlagRequired("project-id")
}
