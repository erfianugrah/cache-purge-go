package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

type Config struct {
	APIToken  string
	APIKey    string
	Email     string
	AccountID string
}

var config Config

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "list":
		handleList()
	case "purge":
		handlePurge(os.Args[2:])
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func handleList() {
	flags := flag.NewFlagSet("list", flag.ExitOnError)
	setupGlobalFlags(flags)
	flags.Parse(os.Args[2:])

	validateAuth()

	client := createClient()

	zones, err := client.ListZones(context.Background())
	if err != nil {
		fmt.Printf("Error listing zones: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nAvailable zones:")
	fmt.Printf("%-40s %-30s %s\n", "Domain", "Zone ID", "Status")
	fmt.Println(strings.Repeat("-", 80))
	for _, zone := range zones {
		fmt.Printf("%-40s %-30s %s\n", zone.Name, zone.ID, zone.Status)
	}
}

func handlePurge(args []string) {
	flags := flag.NewFlagSet("purge", flag.ExitOnError)

	setupGlobalFlags(flags)
	hosts := flags.String("hosts", "", "Comma-separated list of hosts to purge")
	urls := flags.String("urls", "", "Comma-separated list of URLs to purge")
	tags := flags.String("tags", "", "Comma-separated list of cache tags to purge")
	all := flags.Bool("all", false, "Apply to all zones")
	everything := flags.Bool("everything", false, "Purge everything from specified zones")
	quiet := flags.Bool("quiet", false, "Suppress success messages")

	flags.Usage = func() {
		fmt.Println("Usage: cfpurge purge [-flags] <zone...>")
		fmt.Println("\nExamples:")
		fmt.Println("  cfpurge purge -everything example.com")
		fmt.Println("  cfpurge purge -hosts=\"api.example.com,www.example.com\"")
		fmt.Println("  cfpurge purge -urls=\"https://example.com/page1\"")
		fmt.Println("  cfpurge purge -all -tags=\"tag1,tag2\"")
		fmt.Println("\nFlags:")
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	validateAuth()
	client := createClient()

	zoneArgs := flags.Args()
	if len(zoneArgs) == 0 && !*all && *hosts == "" && *urls == "" && *tags == "" {
		fmt.Println("Error: Must specify at least one zone, use -all flag, or provide hosts/urls/tags")
		flags.Usage()
		os.Exit(1)
	}

	zones, err := client.ListZones(context.Background())
	if err != nil {
		fmt.Printf("Error getting zones: %v\n", err)
		os.Exit(1)
	}

	zoneMap := make(map[string]cloudflare.Zone)
	for _, zone := range zones {
		zoneMap[zone.Name] = zone
		zoneMap[zone.ID] = zone
	}

	var targetZones []cloudflare.Zone
	if *all {
		targetZones = zones
	} else if len(zoneArgs) > 0 {
		for _, arg := range zoneArgs {
			if zone, ok := zoneMap[arg]; ok {
				targetZones = append(targetZones, zone)
			} else {
				fmt.Printf("Warning: Zone '%s' not found\n", arg)
			}
		}
	} else if *hosts != "" || *urls != "" {
		hostsList := splitCommaList(*hosts)
		urlsList := splitCommaList(*urls)

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
			fmt.Printf("Error: No matching zones found for the specified hosts/URLs\n")
			fmt.Printf("Available zones:\n")
			for _, zone := range zones {
				fmt.Printf("  %s\n", zone.Name)
			}
			os.Exit(1)
		}
	}

	successCount := 0
	failureCount := 0

	for _, zone := range targetZones {
		if *everything {
			_, err := client.PurgeEverything(context.Background(), zone.ID)
			if err != nil {
				fmt.Printf("❌ Error purging everything from %s: %v\n", zone.Name, err)
				failureCount++
				continue
			}
			if !*quiet {
				fmt.Printf("✅ Successfully purged everything from %s\n", zone.Name)
			}
			successCount++
			continue
		}

		var purgeHosts []string
		var purgeURLs []string

		if *hosts != "" {
			for _, host := range splitCommaList(*hosts) {
				if strings.HasSuffix(host, zone.Name) {
					purgeHosts = append(purgeHosts, host)
				}
			}
		}

		if *urls != "" {
			for _, url := range splitCommaList(*urls) {
				if strings.Contains(url, zone.Name) {
					purgeURLs = append(purgeURLs, url)
				}
			}
		}

		if len(purgeHosts) > 0 || len(purgeURLs) > 0 || *tags != "" {
			var err error

			if len(purgeHosts) > 0 {
				_, err = client.PurgeCache(context.Background(), zone.ID, cloudflare.PurgeCacheRequest{
					Hosts: purgeHosts,
				})
			}

			if len(purgeURLs) > 0 {
				_, err = client.PurgeCache(context.Background(), zone.ID, cloudflare.PurgeCacheRequest{
					Files: purgeURLs,
				})
			}

			if len(splitCommaList(*tags)) > 0 {
				_, err = client.PurgeCache(context.Background(), zone.ID, cloudflare.PurgeCacheRequest{
					Tags: splitCommaList(*tags),
				})
			}
			if err != nil {
				fmt.Printf("❌ Error purging cache for %s: %v\n", zone.Name, err)
				failureCount++
				continue
			}

			if !*quiet {
				if len(purgeHosts) > 0 {
					fmt.Printf("✅ Purged hosts from %s: %s\n", zone.Name, strings.Join(purgeHosts, ", "))
				}
				if len(purgeURLs) > 0 {
					fmt.Printf("✅ Purged URLs from %s: %s\n", zone.Name, strings.Join(purgeURLs, ", "))
				}
				if *tags != "" {
					fmt.Printf("✅ Purged tags from %s: %s\n", zone.Name, *tags)
				}
			}
			successCount++
		}
	}

	fmt.Printf("\nSummary: %d successful, %d failed\n", successCount, failureCount)
	if failureCount > 0 {
		os.Exit(1)
	}
}

func createClient() *cloudflare.API {
	var api *cloudflare.API
	var err error

	if config.APIToken != "" {
		api, err = cloudflare.NewWithAPIToken(config.APIToken)
	} else {
		api, err = cloudflare.New(config.APIKey, config.Email)
	}

	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		os.Exit(1)
	}

	return api
}

func setupGlobalFlags(flags *flag.FlagSet) {
	flags.StringVar(&config.APIToken, "token", os.Getenv("CLOUDFLARE_API_TOKEN"), "Cloudflare API Token")
	flags.StringVar(&config.APIKey, "key", os.Getenv("CLOUDFLARE_API_KEY"), "Cloudflare API Key")
	flags.StringVar(&config.Email, "email", os.Getenv("CLOUDFLARE_EMAIL"), "Cloudflare Email Address")
	flags.StringVar(&config.AccountID, "account", os.Getenv("CLOUDFLARE_ACCOUNT_ID"), "Cloudflare Account ID")
}

func validateAuth() {
	if config.APIToken == "" && (config.APIKey == "" || config.Email == "") {
		fmt.Println("Error: Either API Token or both API Key and Email are required.")
		fmt.Println("\nSet using environment variables:")
		fmt.Println("  CLOUDFLARE_API_TOKEN or (CLOUDFLARE_API_KEY and CLOUDFLARE_EMAIL)")
		fmt.Println("\nOr using flags:")
		fmt.Println("  -token=<token> or (-key=<key> and -email=<email>)")
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: cfpurge <command> [flags] [arguments]")
	fmt.Println("\nAvailable commands:")
	fmt.Println("  list        List all zones in your Cloudflare account")
	fmt.Println("  purge       Purge cache for specified zones")
	fmt.Println("\nUse 'cfpurge <command> -h' for command-specific help")
}

func splitCommaList(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}
