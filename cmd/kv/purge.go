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

func newPurgeCmd() *cobra.Command {
	var (
		deleteByTag   string
		namespace     string
		allNamespaces bool
		dryRun        bool
	)

	cmd := &cobra.Command{
		Use:   "purge",
		Short: "Delete KV entries and purge cache",
		Long:  `Delete Workers KV entries by cache-tag and purge related Cloudflare cache.`,
		Example: `  # Delete entries and purge cache
  cfpurge kv purge --namespace=<namespace-id> --tag=product-123
  
  # Across multiple namespaces
  cfpurge kv purge --namespace=<namespace-id1>,<namespace-id2> --tag=product-123
  
  # Preview what would be deleted (dry run)
  cfpurge kv purge --all-namespaces --tag=product-123 --dry-run`,
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

			if deleteByTag == "" {
				return fmt.Errorf("tag is required for deletion")
			}

			client, err := api.GetClient()
			if err != nil {
				return err
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
			var allCacheTags []string

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
				var cacheTags []string

				for _, key := range keys {
					if key.Metadata != nil {
						// Use type assertion to access the metadata map
						if metadata, ok := key.Metadata.(map[string]interface{}); ok {
							if cacheTag, exists := metadata["cache-tag"]; exists {
								// Check if the cache tag contains our search tag
								if cacheTagStr, ok := cacheTag.(string); ok && strings.Contains(cacheTagStr, deleteByTag) {
									keysToDelete = append(keysToDelete, key.Name)
									cacheTags = append(cacheTags, cacheTagStr)
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
					for i, key := range keysToDelete {
						fmt.Printf("  %s (cache-tag: %s)\n", key, cacheTags[i])
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
				allCacheTags = append(allCacheTags, cacheTags...)
			}

			// Purge the cache with matching cache tags
			if len(allCacheTags) > 0 && !dryRun {
				util.Header("Purging Cloudflare cache with matching cache tags")

				// Get all zones to purge from
				zones, err := client.ListZones(context.Background())
				if err != nil {
					util.Error("Error getting zones for cache purge: %v", err)
				} else {
					// Create unique set of tags
					uniqueTags := util.StringSliceToSet(allCacheTags)

					var tagsList []string
					for tag := range uniqueTags {
						tagsList = append(tagsList, tag)
					}

					util.Info("Found %d unique cache tags to purge", len(tagsList))

					// Purge cache in batches of 30 tags per request
					purgeSuccessCount := 0
					purgeFailureCount := 0

					for i := 0; i < len(tagsList); i += 30 {
						end := i + 30
						if end > len(tagsList) {
							end = len(tagsList)
						}

						batchTags := tagsList[i:end]

						for _, zone := range zones {
							_, err = client.PurgeCache(context.Background(), zone.ID, cloudflare.PurgeCacheRequest{
								Tags: batchTags,
							})

							if err != nil {
								util.Error("Error purging cache for zone %s:%v", zone.Name, err)
								purgeFailureCount++
							} else {
								util.Success("Successfully purged cache tags from zone %s", zone.Name)
								purgeSuccessCount++
							}
						}
					}

					util.PrettyPrintResults(purgeSuccessCount, purgeFailureCount)
				}
			}

			fmt.Printf("\nOverall KV deletion summary: %d successful, %d failed\n", totalSuccessCount, totalFailureCount)
			return nil
		},
	}

	cmd.Flags().StringVar(&deleteByTag, "tag", "", "Delete KV entries with matching cache-tag metadata")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Comma-separated list of KV namespace IDs")
	cmd.Flags().BoolVar(&allNamespaces, "all-namespaces", false, "Apply to all KV namespaces")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be deleted without actually deleting")

	cmd.MarkFlagRequired("tag")

	return cmd
}
