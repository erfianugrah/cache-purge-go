package kv

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"cf-purge/internal/api"

	"github.com/cloudflare/cloudflare-go"
	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	var (
		namespace string
		key       string
		metadata  bool
	)

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a KV entry",
		Long:  `Retrieve a Workers KV entry value or metadata from a namespace.`,
		Example: `  # Get the value of a key
  cfpurge kv get --namespace=<namespace-id> --key=my-key
  
  # Get only the metadata of a key
  cfpurge kv get --namespace=<namespace-id> --key=my-key --metadata`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := api.ValidateAuth(); err != nil {
				return err
			}

			if err := api.ValidateAccountID(); err != nil {
				return err
			}

			if namespace == "" {
				return fmt.Errorf("namespace ID is required")
			}

			if key == "" {
				return fmt.Errorf("key is required")
			}

			client, err := api.GetClient()
			if err != nil {
				return err
			}

			if metadata {
				// Get metadata only
				meta, err := client.GetWorkersKVEntryMetadata(context.Background(), cloudflare.GetWorkersKVEntryMetadataParams{
					NamespaceID: namespace,
					AccountID:   api.GetAccountID(),
					Key:         key,
				})

				if err != nil {
					return fmt.Errorf("error getting KV metadata: %w", err)
				}

				fmt.Println("KV Entry Metadata:")
				if metaData, ok := meta.(map[string]interface{}); ok {
					for k, v := range metaData {
						fmt.Printf("  %s: %v\n", k, v)
					}
				} else if meta == nil {
					fmt.Println("  No metadata found")
				} else {
					metadata, _ := json.MarshalIndent(meta, "", "  ")
					fmt.Println(string(metadata))
				}
			} else {
				// Get value
				value, err := client.GetWorkersKV(context.Background(), cloudflare.GetWorkersKVParams{
					NamespaceID: namespace,
					AccountID:   api.GetAccountID(),
					Key:         key,
				})

				if err != nil {
					return fmt.Errorf("error getting KV value: %w", err)
				}

				// Try to print as string first
				valueStr := string(value)
				if strings.HasPrefix(valueStr, "{") || strings.HasPrefix(valueStr, "[") {
					// If it looks like JSON, pretty print it
					var jsonValue interface{}
					if err := json.Unmarshal(value, &jsonValue); err == nil {
						prettyJSON, _ := json.MarshalIndent(jsonValue, "", "  ")
						fmt.Println(string(prettyJSON))
					} else {
						fmt.Println(valueStr)
					}
				} else {
					fmt.Println(valueStr)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&namespace, "namespace", "", "KV namespace ID")
	cmd.Flags().StringVar(&key, "key", "", "Key to retrieve")
	cmd.Flags().BoolVar(&metadata, "metadata", false, "Show metadata only (not value)")

	cmd.MarkFlagRequired("namespace")
	cmd.MarkFlagRequired("key")

	return cmd
}
