package kv

import (
	"context"
	"fmt"

	"cf-purge/internal/api"
	"cf-purge/internal/util"

	"github.com/cloudflare/cloudflare-go"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	var title string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new KV namespace",
		Long:  `Create a new Workers KV namespace in your Cloudflare account.`,
		Example: `  # Create a new namespace
  cfpurge kv create --title="My Namespace"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := api.ValidateAuth(); err != nil {
				return err
			}

			if err := api.ValidateAccountID(); err != nil {
				return err
			}

			if title == "" {
				return fmt.Errorf("namespace title is required")
			}

			client, err := api.GetClient()
			if err != nil {
				return err
			}

			// Create KV namespace
			res, err := client.CreateWorkersKVNamespace(
				context.Background(),
				api.GetAccountID(),
				cloudflare.CreateWorkersKVNamespaceParams{
					Title: title,
				},
			)

			if err != nil {
				return fmt.Errorf("error creating KV namespace: %w", err)
			}

			util.Success("Successfully created KV namespace: %s", title)
			fmt.Printf("   Namespace ID: %s\n", res.ID)

			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Title for the new KV namespace")
	cmd.MarkFlagRequired("title")

	return cmd
}
