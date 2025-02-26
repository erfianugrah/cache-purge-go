package kv

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"cf-purge/internal/api"
	"cf-purge/internal/util"

	"github.com/cloudflare/cloudflare-go"
	"github.com/spf13/cobra"
)

func newPutCmd() *cobra.Command {
	var (
		namespace      string
		key            string
		value          string
		valueFile      string
		expirationTTL  int
		expirationDate string
		cacheTag       string
		metadata       string
	)

	cmd := &cobra.Command{
		Use:   "put",
		Short: "Put a KV entry",
		Long:  `Create or update a Workers KV entry in a namespace.`,
		Example: `  # Store a simple value
  cfpurge kv put --namespace=<namespace-id> --key=my-key --value="my value"
  
  # Store value from a file with cache tag
  cfpurge kv put --namespace=<namespace-id> --key=my-key --file=data.json --cache-tag=product-123
  
  # With expiration
  cfpurge kv put --namespace=<namespace-id> --key=my-key --value="temp" --ttl=3600`,
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

			if value == "" && valueFile == "" {
				return fmt.Errorf("either value or file is required")
			}

			// Parse metadata if provided
			var metadataMap map[string]interface{}
			if metadata != "" {
				if err := json.Unmarshal([]byte(metadata), &metadataMap); err != nil {
					return fmt.Errorf("error parsing metadata JSON: %w", err)
				}
			}

			// Add cache tag to metadata if provided
			if cacheTag != "" {
				if metadataMap == nil {
					metadataMap = make(map[string]interface{})
				}
				metadataMap["cache-tag"] = cacheTag
			}

			client, err := api.GetClient()
			if err != nil {
				return err
			}

			// Get the value data
			var valueData []byte
			if valueFile != "" {
				data, err := os.ReadFile(valueFile)
				if err != nil {
					return fmt.Errorf("error reading file: %w", err)
				}
				valueData = data
			} else {
				valueData = []byte(value)
			}

			// Prepare expiration
			var expiration *time.Time
			if expirationDate != "" {
				parsedTime, err := time.Parse(time.RFC3339, expirationDate)
				if err != nil {
					return fmt.Errorf("error parsing expiration date (should be RFC3339 format): %w", err)
				}
				expiration = &parsedTime
			}

			// Create params for write
			params := cloudflare.WriteWorkersKVEntryParams{
				NamespaceID: namespace,
				AccountID:   api.GetAccountID(),
				Key:         key,
				Value:       valueData,
			}

			// Add metadata and expiration if provided
			if metadataMap != nil {
				params.Metadata = metadataMap
			}

			if expirationTTL > 0 {
				ttl := uint(expirationTTL)
				params.ExpirationTTL = &ttl
			} else if expiration != nil {
				// Convert to seconds since epoch
				expSeconds := uint(expiration.Unix())
				params.Expiration = &expSeconds
			}

			// Write the KV entry
			err = client.WriteWorkersKVEntry(context.Background(), params)
			if err != nil {
				return fmt.Errorf("error writing KV entry: %w", err)
			}

			util.Success("Successfully stored value for key: %s", key)

			// Print details about the entry
			if metadataMap != nil {
				fmt.Println("   With metadata:")
				for k, v := range metadataMap {
					fmt.Printf("     %s: %v\n", k, v)
				}
			}

			if expirationTTL > 0 {
				fmt.Printf("   Expiration: %d seconds (TTL)\n", expirationTTL)
			} else if expiration != nil {
				fmt.Printf("   Expiration: %s\n", expiration.Format(time.RFC3339))
			} else {
				fmt.Println("   No expiration set")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&namespace, "namespace", "", "KV namespace ID")
	cmd.Flags().StringVar(&key, "key", "", "Key to create or update")
	cmd.Flags().StringVar(&value, "value", "", "Value to store")
	cmd.Flags().StringVar(&valueFile, "file", "", "Read value from file")
	cmd.Flags().IntVar(&expirationTTL, "ttl", 0, "Expiration TTL in seconds (0 = no expiration)")
	cmd.Flags().StringVar(&expirationDate, "expiration", "", "Expiration date/time (RFC3339 format)")
	cmd.Flags().StringVar(&cacheTag, "cache-tag", "", "Cache tag for the entry")
	cmd.Flags().StringVar(&metadata, "metadata", "", "Custom metadata JSON (e.g., '{\"key\":\"value\"}')")

	cmd.MarkFlagRequired("namespace")
	cmd.MarkFlagRequired("key")

	return cmd
}
