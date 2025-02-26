package kv

import (
	"github.com/spf13/cobra"
)

// NewKVCmd creates the kv command and its subcommands
func NewKVCmd() *cobra.Command {
	kvCmd := &cobra.Command{
		Use:   "kv",
		Short: "Manage Cloudflare Workers KV",
		Long:  `Manage Cloudflare Workers KV namespaces and entries.`,
	}

	// Add all KV subcommands
	kvCmd.AddCommand(newListCmd())
	kvCmd.AddCommand(newCreateCmd())
	kvCmd.AddCommand(newDeleteCmd())
	kvCmd.AddCommand(newPurgeCmd())
	kvCmd.AddCommand(newGetCmd())
	kvCmd.AddCommand(newPutCmd())
	kvCmd.AddCommand(newRenameCmd())

	return kvCmd
}
