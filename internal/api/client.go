package api

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
)

// Config holds API credentials and configuration
type Config struct {
	APIToken  string
	APIKey    string
	Email     string
	AccountID string
}

var config Config

// SetConfig updates the global API configuration
func SetConfig(cfg Config) {
	config = cfg
}

// GetClient creates a new Cloudflare API client
func GetClient() (*cloudflare.API, error) {
	var api *cloudflare.API
	var err error

	if config.APIToken != "" {
		api, err = cloudflare.NewWithAPIToken(config.APIToken)
	} else if config.APIKey != "" && config.Email != "" {
		api, err = cloudflare.New(config.APIKey, config.Email)
	} else {
		return nil, fmt.Errorf("either API Token or both API Key and Email are required")
	}

	if err != nil {
		return nil, fmt.Errorf("error creating Cloudflare client: %w", err)
	}

	return api, nil
}

// ValidateAuth checks if authentication credentials are valid
func ValidateAuth() error {
	if config.APIToken == "" && (config.APIKey == "" || config.Email == "") {
		return fmt.Errorf("either API Token or both API Key and Email are required")
	}
	return nil
}

// ValidateAccountID checks if account ID is provided for operations that require it
func ValidateAccountID() error {
	if config.AccountID == "" {
		return fmt.Errorf("Cloudflare Account ID is required for this operation")
	}
	return nil
}

// GetAccountID returns the configured account ID
func GetAccountID() string {
	return config.AccountID
}

// ListZones gets all zones for the account
func ListZones(ctx context.Context) ([]cloudflare.Zone, error) {
	client, err := GetClient()
	if err != nil {
		return nil, err
	}

	zones, err := client.ListZones(ctx)
	if err != nil {
		return nil, err
	}

	return zones, nil
}
