package cmd

import (
	"fmt"
	"os"

	"cfpurge/cmd/kv"
	"cfpurge/internal/api"

	"github.com/spf13/cobra"
)

var (
	cfgAPIToken  string
	cfgAPIKey    string
	cfgEmail     string
	cfgAccountID string

	version   string
	buildTime string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cfpurge",
	Short: "Cloudflare cache purge and KV management CLI tool",
	Long: `A command-line tool for managing Cloudflare cache purge operations and Workers KV.
Supports purging by hosts, URLs, tags, and everything across zones,
as well as complete management of Workers KV namespaces and entries.`,
	Version: version,
}

// SetVersionInfo sets the version information for the root command
func SetVersionInfo(v, bt string) {
	version = v
	buildTime = bt
	rootCmd.Version = fmt.Sprintf("%s (built at %s)", version, buildTime)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgAPIToken, "token", os.Getenv("CLOUDFLARE_API_TOKEN"), "Cloudflare API Token")
	rootCmd.PersistentFlags().StringVar(&cfgAPIKey, "key", os.Getenv("CLOUDFLARE_API_KEY"), "Cloudflare API Key")
	rootCmd.PersistentFlags().StringVar(&cfgEmail, "email", os.Getenv("CLOUDFLARE_EMAIL"), "Cloudflare Email Address")
	rootCmd.PersistentFlags().StringVar(&cfgAccountID, "account", os.Getenv("CLOUDFLARE_ACCOUNT_ID"), "Cloudflare Account ID")

	// Add commands
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(purgeCmd)
	rootCmd.AddCommand(kv.NewKVCmd())
}

// initConfig sets up the config based on flags and environment variables
func initConfig() {
	// Set up API client configuration
	api.SetConfig(api.Config{
		APIToken:  cfgAPIToken,
		APIKey:    cfgAPIKey,
		Email:     cfgEmail,
		AccountID: cfgAccountID,
	})
}
