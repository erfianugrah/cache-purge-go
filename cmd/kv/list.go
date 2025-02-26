package kv

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cf-purge/internal/api"

	"github.com/cloudflare/cloudflare-go"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var namespace string
	var verbose bool
	var filter string
	var limit int
	var cursor string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List KV namespaces or keys in a namespace",
		Long: `List all KV namespaces in your Cloudflare account,
or list keys in a specific namespace.`,
		Example: `  # List all namespaces
  cfpurge kv list
  
  # List keys in a namespace
  cfpurge kv list --namespace=<namespace-id>
  
  # List keys with metadata and filtering
  cfpurge kv list --namespace=<namespace-id> --verbose --filter=user- --limit=50`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := api.ValidateAuth(); err != nil {
				return err
			}

			if err := api.ValidateAccountID(); err != nil {
				return err
			}

			client, err := api.GetClient()
			if err != nil {
				return err
			}

			// If no namespace provided, list all namespaces
			if namespace == "" {
				return listNamespaces(client)
			}

			// List keys in the namespace
			return listKeys(client, namespace, verbose, filter, limit, cursor)
		},
	}

	cmd.Flags().StringVar(&namespace, "namespace", "", "KV namespace ID to list keys from")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Display key metadata")
	cmd.Flags().StringVar(&filter, "filter", "", "Filter keys by prefix")
	cmd.Flags().IntVar(&limit, "limit", 1000, "Maximum number of keys to return")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Cursor for pagination")

	return cmd
}

func listNamespaces(client *cloudflare.API) error {
	namespaces, _, err := client.ListWorkersKVNamespaces(context.Background(), api.GetAccountID(), cloudflare.ListWorkersKVNamespacesParams{})
	if err != nil {
		return fmt.Errorf("error listing KV namespaces: %w", err)
	}

	fmt.Println("\nAvailable KV namespaces:")
	fmt.Printf("%-40s %-30s\n", "Title", "Namespace ID")
	fmt.Println(strings.Repeat("-", 80))
	for _, ns := range namespaces {
		fmt.Printf("%-40s %-30s\n", ns.Title, ns.ID)
	}
	return nil
}

func listKeys(client *cloudflare.API, namespace string, verbose bool, filter string, limit int, cursor string) error {
	params := cloudflare.ListWorkersKVKeysParams{
		NamespaceID: namespace,
		AccountID:   api.GetAccountID(),
		Limit:       limit,
	}

	if filter != "" {
		params.Prefix = filter
	}

	if cursor != "" {
		params.Cursor = cursor
	}

	// If verbose is enabled, we need to fetch metadata
	if verbose {
		params.Metadata = true
	}

	keys, listResult, err := client.ListWorkersKVKeys(context.Background(), params)
	if err != nil {
		return fmt.Errorf("error listing KV keys: %w", err)
	}

	fmt.Printf("\nKeys in namespace %s:\n", namespace)
	if verbose {
		fmt.Printf("%-40s %-20s %s\n", "Key", "Expiration", "Metadata")
		fmt.Println(strings.Repeat("-", 80))
		for _, key := range keys {
			expiration := "Never"
			if key.Expiration > 0 {
				expTime := time.Unix(int64(key.Expiration), 0)
				expiration = expTime.Format("2006-01-02 15:04:05")
			}
			metadataStr := "None"
			if key.Metadata != nil {
				metadataBytes, _ := json.MarshalIndent(key.Metadata, "", "  ")
				metadataStr = string(metadataBytes)
			}
			fmt.Printf("%-40s %-20s %s\n", key.Name, expiration, metadataStr)
		}
	} else {
		for _, key := range keys {
			fmt.Println(key.Name)
		}
	}

	// Show pagination information if cursor is available
	if listResult.Result_info.Cursor != "" && listResult.Result_info.Cursor != "null" {
		fmt.Printf("\nMore keys available. Use this cursor for the next page:\n")
		fmt.Printf("  --cursor=%s\n", listResult.Result_info.Cursor)
	}

	fmt.Printf("\nShowing %d/%d keys\n", len(keys), listResult.Result_info.Count)
	return nil
}
