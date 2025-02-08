# Cloudflare Cache Purge CLI

A command-line tool for managing Cloudflare cache purging across multiple zones with automatic subdomain matching.

## Features

- List all zones in your Cloudflare account
- Purge cache by hosts, URLs, or tags
- Automatic subdomain matching to parent domains
- Support for both API Token and API Key authentication
- Batch purging across multiple zones
- Clear success/error reporting

## Installation

```bash
git clone https://github.com/erfianugrah/cache-purge-go
cd cache-purge-go
make build
```

## Configuration

Set your Cloudflare credentials using environment variables:

```bash
# Using API Token (recommended)
export CLOUDFLARE_API_TOKEN="your-api-token"

# Or using API Key + Email
export CLOUDFLARE_API_KEY="your-api-key"
export CLOUDFLARE_EMAIL="your-email"

# Optional
export CLOUDFLARE_ACCOUNT_ID="your-account-id"
```

## Usage

List zones:
```bash
cfpurge list
```

Purge by hosts:
```bash
# Automatically matches subdomains to parent domains
cfpurge purge -hosts="api.example.com,cdn.example.com,api.other.com"
```

Purge by URLs:
```bash
cfpurge purge -urls="https://example.com/page1,https://other.com/page1"
```

Purge by tags:
```bash
cfpurge purge -tags="tag1,tag2"
```

Purge everything from specific zones:
```bash
cfpurge purge example.com other.com -everything
```

Purge across all zones:
```bash
cfpurge purge -all -tags="tag1,tag2"
```

## Development

Build the project:
```bash
make build
```

Run tests:
```bash
make test
```

Clean build artifacts:
```bash
make clean
```

## License

MIT License
