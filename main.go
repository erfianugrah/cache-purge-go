package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

const (
	baseURL = "https://api.cloudflare.com/client/v4"
)

type Config struct {
	APIToken  string
	APIKey    string
	Email     string
	AccountID string
}

type PurgeRequest struct {
	Files           []string `json:"files,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	Hosts           []string `json:"hosts,omitempty"`
	PurgeEverything bool     `json:"purge_everything,omitempty"`
}

type Zone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type ZoneListResponse struct {
	Success bool    `json:"success"`
	Errors  []Error `json:"errors"`
	Result  []Zone  `json:"result"`
}

type CloudflareResponse struct {
	Success bool     `json:"success"`
	Errors  []Error  `json:"errors"`
	Result  struct{} `json:"result"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ZonePurgeConfig struct {
	Zone  Zone
	Hosts []string
	URLs  []string
	Tags  []string
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

	zones, err := listAllZones()
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

	// Custom usage message
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

	// Parse flags
	if err := flags.Parse(args); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	validateAuth()

	// Get non-flag arguments (zones)
	zoneArgs := flags.Args()

	if len(zoneArgs) == 0 && !*all && *hosts == "" && *urls == "" && *tags == "" {
		fmt.Println("Error: Must specify at least one zone, use -all flag, or provide hosts/urls/tags")
		flags.Usage()
		os.Exit(1)
	}

	zones, err := listAllZones()
	if err != nil {
		fmt.Printf("Error getting zones: %v\n", err)
		os.Exit(1)
	}

	// Create zone maps for lookups
	zoneMap := make(map[string]Zone)       // For exact matches
	zonesByDomain := make(map[string]Zone) // For domain matching
	for _, zone := range zones {
		zoneMap[zone.Name] = zone
		zoneMap[zone.ID] = zone
		zonesByDomain[zone.Name] = zone
	}

	// Handle specific zones or all zones
	var targetZones []Zone
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
	}

	// Process purge for each target zone
	for _, zone := range targetZones {
		if *everything {
			req := PurgeRequest{
				PurgeEverything: true,
			}
			if err := purgeCacheAPI(zone.ID, req); err != nil {
				fmt.Printf("Error purging everything from %s: %v\n", zone.Name, err)
				continue
			}
			fmt.Printf("Successfully purged everything from %s\n", zone.Name)
			continue
		}

		var purgeHosts []string
		var purgeURLs []string

		// Match hosts to this zone
		if *hosts != "" {
			for _, host := range splitCommaList(*hosts) {
				if strings.HasSuffix(host, zone.Name) {
					purgeHosts = append(purgeHosts, host)
				}
			}
		}

		// Match URLs to this zone
		if *urls != "" {
			for _, url := range splitCommaList(*urls) {
				if strings.Contains(url, zone.Name) {
					purgeURLs = append(purgeURLs, url)
				}
			}
		}

		// Create and send purge request if needed
		if len(purgeHosts) > 0 || len(purgeURLs) > 0 || *tags != "" {
			req := PurgeRequest{
				Files: purgeURLs,
				Tags:  splitCommaList(*tags),
				Hosts: purgeHosts,
			}

			if err := purgeCacheAPI(zone.ID, req); err != nil {
				fmt.Printf("Error purging cache for %s: %v\n", zone.Name, err)
				continue
			}

			// Print what was purged
			if len(purgeHosts) > 0 {
				fmt.Printf("Purged hosts from %s: %s\n", zone.Name, strings.Join(purgeHosts, ", "))
			}
			if len(purgeURLs) > 0 {
				fmt.Printf("Purged URLs from %s: %s\n", zone.Name, strings.Join(purgeURLs, ", "))
			}
			if *tags != "" {
				fmt.Printf("Purged tags from %s: %s\n", zone.Name, *tags)
			}
		}
	}
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

func listAllZones() ([]Zone, error) {
	url := fmt.Sprintf("%s/zones", baseURL)
	if config.AccountID != "" {
		url += fmt.Sprintf("?account.id=%s", config.AccountID)
	}

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	setAuthHeaders(request)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var zoneResponse ZoneListResponse
	if err := json.Unmarshal(body, &zoneResponse); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if !zoneResponse.Success {
		var errMsgs []string
		for _, err := range zoneResponse.Errors {
			errMsgs = append(errMsgs, err.Message)
		}
		return nil, fmt.Errorf("cloudflare API errors: %s", strings.Join(errMsgs, "; "))
	}

	return zoneResponse.Result, nil
}

func purgeCacheAPI(zoneID string, req PurgeRequest) error {
	url := fmt.Sprintf("%s/zones/%s/purge_cache", baseURL, zoneID)

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshaling request: %v", err)
	}

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	setAuthHeaders(request)
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer response.Body.Close()

	var cfResponse CloudflareResponse
	if err := json.NewDecoder(response.Body).Decode(&cfResponse); err != nil {
		return fmt.Errorf("error parsing response: %v", err)
	}

	if !cfResponse.Success {
		return fmt.Errorf("cloudflare API error")
	}

	return nil
}

func setAuthHeaders(request *http.Request) {
	if config.APIToken != "" {
		request.Header.Set("Authorization", "Bearer "+config.APIToken)
	} else {
		request.Header.Set("X-Auth-Key", config.APIKey)
		request.Header.Set("X-Auth-Email", config.Email)
	}
}
