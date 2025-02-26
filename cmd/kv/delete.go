package kv

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"cf-purge/internal/api"
	"cf-purge/internal/util"

	"github.com/cloudflare/cloudflare-go"
	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	var (
		deleteByTag   string
		namespace     string
		allNamespaces bool
		key           string
		dryRun        bool
	)

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete KV entries",
		Long:  `Delete Workers KV entries by key or by matching cache-tag metadata.`,
		Example: `  # Delete a specific key
  cfpurge kv delete --namespace=<namespace-id> --key=my-key
  
  # Delete entries with matching tag
  cfpurge kv delete --namespace=<namespace-id> --tag=product-123
  
  # Delete entries across multiple namespaces
  cfpurge kv delete --namespace=<namespace-id1>,<namespace-id2> --tag=product-123
  
  # Delete entries in all namespaces
  cfpurge kv delete --all-namespaces --tag=product-123
  
  # Preview what would be deleted (dry run)
  cfpurge kv delete --namespace=<namespace-id> --tag=product-123 --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := api.ValidateAuth(); err != nil {
				return err
			}

			if err := api.ValidateAccountID(); err != nil {
				return err
			}

			if namespace == "" && !allNamespaces {
				return fmt.Errorf("either namespace ID or --all-namespaces flag is required")
			}

			if deleteByTag == "" && key == "" {
				return fmt.Errorf("either tag or key is required for deletion")
			}

			client, err := api.GetClient()
			if err != nil {
				return err
			}

			// If deleting a specific key, handle it directly
			if key != "" {
				if allNamespaces {
					return fmt.Errorf("cannot use --all-namespaces with --key; specify a single namespace")
				}

				// Split namespaces if multiple are provided
				namespaces := util.SplitCommaList(namespace)
				if len(namespaces) > 1 {
					return fmt.Errorf("cannot use multiple namespaces with --key; specify a single namespace")
				}

				if dryRun {
					util.Info("Dry run mode - would delete key '%s' from namespace %s", key, namespaces[0])
					return nil
				}

				err := client.DeleteWorkersKVEntry(context.Background(), cloudflare.DeleteWorkersKVEntryParams{
					NamespaceID: namespaces[0],
					AccountID:   api.GetAccountID(),
					Key:         key,
				})

				if err != nil {
					return fmt.Errorf("error deleting KV key: %w", err)
				}

				util.Success("Successfully deleted key: %s", key)
				return nil
			}

			// Get list of namespaces to process
			var namespaceIDs []string

			if allNamespaces {
				// Get all namespaces
				namespaces, _, err := client.ListWorkersKVNamespaces(context.Background(), api.GetAccountID(), cloudflare.ListWorkersKVNamespacesParams{})
				if err != nil {
					return fmt.Errorf("error listing KV namespaces: %w", err)
				}

				if len(namespaces) == 0 {
					return fmt.Errorf("no KV namespaces found in account")
				}

				util.Info("Found %d KV namespaces to process", len(namespaces))
				for _, ns := range namespaces {
					namespaceIDs = append(namespaceIDs, ns.ID)
				}
			} else {
				// Use provided namespace IDs
				namespaceIDs = util.SplitCommaList(namespace)
			}

			totalSuccessCount := 0
			totalFailureCount := 0

			// Process each namespace
			for _, nsID := range namespaceIDs {
				fmt.Printf("\nProcessing namespace: %s\n", nsID)

				// Get all keys in the namespace
				keys, _, err := client.ListWorkersKVKeys(context.Background(), cloudflare.ListWorkersKVKeysParams{
					NamespaceID: nsID,
					AccountID:   api.GetAccountID(),
					Metadata:    true,
				})
				if err != nil {
					util.Error("Error listing KV keys in namespace %s: %v", nsID, err)
					totalFailureCount++
					continue
				}

				// Find keys with matching cache tags
				var keysToDelete []string

				for _, key := range keys {
					if key.Metadata != nil {
						// Use type assertion to access the metadata map
						if metadata, ok := key.Metadata.(map[string]interface{}); ok {
							if cacheTag, exists := metadata["cache-tag"]; exists {
								// Check if the cache tag contains our search tag
								if cacheTagStr, ok := cacheTag.(string); ok && strings.Contains(cacheTagStr, deleteByTag) {
									keysToDelete = append(keysToDelete, key.Name)
								}
							}
						}
					}
				}

				if len(keysToDelete) == 0 {
					util.Info("No KV keys found with cache-tag containing '%s' in namespace %s", deleteByTag, nsID)
					continue
				}

				util.Info("Found %d KV keys with matching cache tag '%s' in namespace %s", len(keysToDelete), deleteByTag, nsID)

				if dryRun {
					fmt.Printf("Dry run mode - would delete the following keys from namespace %s:\n", nsID)
					for _, key := range keysToDelete {
						fmt.Printf("  %s\n", key)
					}
					continue
				}

				// Delete the KV entries
				var wg sync.WaitGroup
				var deleteMutex sync.Mutex
				successCount := 0
				failureCount := 0

				// Process in batches of 30 for better performance
				batchSize := 30
				for i := 0; i < len(keysToDelete); i += batchSize {
					end := i + batchSize
					if end > len(keysToDelete) {
						end = len(keysToDelete)
					}

					batch := keysToDelete[i:end]
					wg.Add(1)

					go func(keys []string, nsID string) {
						defer wg.Done()

						for _, key := range keys {
							err := client.DeleteWorkersKVEntry(context.Background(), cloudflare.DeleteWorkersKVEntryParams{
								NamespaceID: nsID,
								AccountID:   api.GetAccountID(),
								Key:         key,
							})

							deleteMutex.Lock()
							if err != nil {
								util.Error("Error deleting KV key %s in namespace %s: %v", key, nsID, err)
								failureCount++
							} else {
								util.Success("Successfully deleted KV key: %s from namespace %s", key, nsID)
								successCount++
							}
							deleteMutex.Unlock()
						}
					}(batch, nsID)
				}

				// Wait for all KV deletions to complete
				wg.Wait()

				fmt.Printf("Summary for namespace %s: %d successful, %d failed\n", nsID, successCount, failureCount)
				totalSuccessCount += successCount
				totalFailureCount += failureCount
			}

			util.PrettyPrintResults(totalSuccessCount, totalFailureCount)
			return nil
		},
	}

	cmd.Flags().StringVar(&deleteByTag, "tag", "", "Delete KV entries with matching cache-tag metadata")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Comma-separated list of KV namespace IDs")
	cmd.Flags().BoolVar(&allNamespaces, "all-namespaces", false, "Apply to all KV namespaces")
	cmd.Flags().StringVar(&key, "key", "", "Specific key to delete")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be deleted without actually deleting")

	return cmd
}
