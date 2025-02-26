package cmd

import (
	"context"
	"fmt"
	"strings"

	"cfpurge/internal/api"
	"cfpurge/internal/util"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available Cloudflare zones",
	Long:  `List all zones in your Cloudflare account.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := api.ValidateAuth(); err != nil {
			return err
		}

		zones, err := api.ListZones(context.Background())
		if err != nil {
			return fmt.Errorf("error listing zones: %w", err)
		}

		fmt.Println("\nAvailable zones:")
		fmt.Printf("%-40s %-30s %s\n", "Domain", "Zone ID", "Status")
		fmt.Println(strings.Repeat("-", 80))
		for _, zone := range zones {
			fmt.Printf("%-40s %-30s %s\n", zone.Name, zone.ID, zone.Status)
		}

		return nil
	},
}
