package cmd

import (
	"context"
	"fmt"
	"strings"

	"cfpurge/internal/api"
	"cfpurge/internal/util"

	"github.com/cloudflare/cloudflare-go"
	"github.com/spf13/cobra"
)

var (
	purgeHosts      string
	purgeURLs       string
	purgeTags       string
	purgeAll        bool
	purgeEverything bool
	purgeQuiet      bool
)

// purgeCmd represents the purge command
var purgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Purge Cloudflare cache",
	Long:  `Purge cache for specified zones by hosts, URLs, or tags.`,
	Example: `  # Purge everything from a zone
  cfpurge purge --everything example.com
  
  # Purge specific hosts across all zones
  cfpurge purge --all --hosts="api.example.com,www.example.com"
  
  # Purge specific URLs from a zone
  cfpurge purge --urls="https://example.com/page1" example.com`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := api.ValidateAuth(); err != nil {
			return err
		}

		client, err := api.GetClient()
		if err != nil {
			return err
		}

		zoneArgs := args
		if len(zoneArgs) == 0 && !purgeAll && purgeHosts == "" && purgeURLs == "" && purgeTags == "" {
			return fmt.Errorf("must specify at least one zone, use --all flag, or provide hosts/urls/tags")
		}

		zones, err := api.ListZones(context.Background())
		if err != nil {
			return fmt.Errorf("error getting zones: %w", err)
		}

		zoneMap := make(map[string]cloudflare.Zone)
		for _, zone := range zones {
			zoneMap[zone.Name] = zone
			zoneMap[zone.ID] = zone
		}

		var targetZones []cloudflare.Zone
		if purgeAll {
			targetZones = zones
		} else if len(zoneArgs) > 0 {
			for _, arg := range zoneArgs {
				if zone, ok := zoneMap[arg]; ok {
					targetZones = append(targetZones, zone)
				} else {
					util.Warning("Zone '%s' not found", arg)
				}
			}
		} else if purgeHosts != "" || purgeURLs != "" {
			hostsList := util.SplitCommaList(purgeHosts)
			urlsList := util.SplitCommaList(purgeURLs)

			for _, zone := range zones {
				shouldInclude := false

				for _, host := range hostsList {
					if strings.HasSuffix(host, zone.Name) {
						shouldInclude = true
						break
					}
				}

				for _, url := range urlsList {
					if strings.Contains(url, zone.Name) {
						shouldInclude = true
						break
					}
				}

				if shouldInclude {
					targetZones = append(targetZones, zone)
				}
			}

			if len(targetZones) == 0 {
				return fmt.Errorf("no matching zones found for the specified hosts/URLs")
			}
		}

		successCount := 0
		failureCount := 0

		for _, zone := range targetZones {
			if purgeEverything {
				_, err := client.PurgeEverything(context.Background(), zone.ID)
				if err != nil {
					util.Error("Error purging everything from %s: %v", zone.Name, err)
					failureCount++
					continue
				}
				if !purgeQuiet {
					util.Success("Successfully purged everything from %s", zone.Name)
				}
				successCount++
				continue
			}

			var purgeHostsList []string
			var purgeURLsList []string

			if purgeHosts != "" {
				for _, host := range util.SplitCommaList(purgeHosts) {
					if strings.HasSuffix(host, zone.Name) {
						purgeHostsList = append(purgeHostsList, host)
					}
				}
			}

			if purgeURLs != "" {
				for _, url := range util.SplitCommaList(purgeURLs) {
					if strings.Contains(url, zone.Name) {
						purgeURLsList = append(purgeURLsList, url)
					}
				}
			}

			if len(purgeHostsList) > 0 || len(purgeURLsList) > 0 || purgeTags != "" {
				var err error

				if len(purgeHostsList) > 0 {
					purgeReq := cloudflare.PurgeCacheRequest{
						Hosts: purgeHostsList,
					}
					_, err = client.PurgeCache(context.Background(), zone.ID, purgeReq)
				}

				if len(purgeURLsList) > 0 {
					purgeReq := cloudflare.PurgeCacheRequest{
						Files: purgeURLsList,
					}
					_, err = client.PurgeCache(context.Background(), zone.ID, purgeReq)
				}

				if purgeTags != "" {
					// Split tags into batches of 30 (Cloudflare's limit)
					tagsList := util.SplitCommaList(purgeTags)
					for i := 0; i < len(tagsList); i += 30 {
						end := i + 30
						if end > len(tagsList) {
							end = len(tagsList)
						}

						batchTags := tagsList[i:end]
						purgeReq := cloudflare.PurgeCacheRequest{
							Tags: batchTags,
						}
						_, err = client.PurgeCache(context.Background(), zone.ID, purgeReq)

						if err != nil {
							break
						}
					}
				}

				if err != nil {
					util.Error("Error purging cache for %s: %v", zone.Name, err)
					failureCount++
					continue
				}

				if !purgeQuiet {
					if len(purgeHostsList) > 0 {
						util.Success("Purged hosts from %s: %s", zone.Name, strings.Join(purgeHostsList, ", "))
					}
					if len(purgeURLsList) > 0 {
						util.Success("Purged URLs from %s: %s", zone.Name, strings.Join(purgeURLsList, ", "))
					}
					if purgeTags != "" {
						util.Success("Purged tags from %s: %s", zone.Name, purgeTags)
					}
				}
				successCount++
			}
		}

		util.PrettyPrintResults(successCount, failureCount)
		return nil
	},
}

func init() {
	purgeCmd.Flags().StringVar(&purgeHosts, "hosts", "", "Comma-separated list of hosts to purge")
	purgeCmd.Flags().StringVar(&purgeURLs, "urls", "", "Comma-separated list of URLs to purge")
	purgeCmd.Flags().StringVar(&purgeTags, "tags", "", "Comma-separated list of cache tags to purge (Enterprise only)")
	purgeCmd.Flags().BoolVar(&purgeAll, "all", false, "Apply to all zones")
	purgeCmd.Flags().BoolVar(&purgeEverything, "everything", false, "Purge everything from cache")
	purgeCmd.Flags().BoolVar(&purgeQuiet, "quiet", false, "Suppress success messages")
}
