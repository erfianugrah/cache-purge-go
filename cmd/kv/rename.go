package kv

import (
	"context"
	"fmt"

	"cfpurge/internal/api"
	"cfpurge/internal/util"

	"github.com/cloudflare/cloudflare-go"
	"github.com/spf13/cobra"
)

func newRenameCmd() *cobra.Command {
	var (
		namespaceID string
		title       string
	)

	cmd := &cobra.Command{
		Use:   "rename",
		Short: "Rename a KV namespace",
		Long:  `Change the title of an existing Workers KV namespace.`,
		Example: `  # Rename a namespace
  cfpurge kv rename --namespace=<namespace-id> --title="New Name"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := api.ValidateAuth(); err != nil {
				return err
			}

			if err := api.ValidateAccountID(); err != nil {
				return err
			}

			if namespaceID == "" {
				return fmt.Errorf("namespace ID is required")
			}

			if title == "" {
				return fmt.Errorf("new title is required")
			}

			client, err := api.GetClient()
			if err != nil {
				return err
			}

			// Rename KV namespace
			params := cloudflare.UpdateWorkersKVNamespaceParams{
				NamespaceID: namespaceID,
				Title:       title,
			}
			_, err = client.UpdateWorkersKVNamespace(
				context.Background(),
				api.GetAccountID(),
				params,
			)

			if err != nil {
				return fmt.Errorf("error renaming KV namespace: %w", err)
			}

			util.Success("Successfully renamed namespace to: %s", title)
			return nil
		},
	}

	cmd.Flags().StringVar(&namespaceID, "namespace", "", "KV namespace ID to rename")
	cmd.Flags().StringVar(&title, "title", "", "New title for the KV namespace")

	cmd.MarkFlagRequired("namespace")
	cmd.MarkFlagRequired("title")

	return cmd
}
