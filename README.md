# Cloudflare Cache Purge CLI Tool

A command-line tool for managing Cloudflare cache purge operations across zones.

## Features

- List all zones in your Cloudflare account
- Purge cache by hosts
- Purge cache by URLs
- Purge cache by tags (Enterprise only)
- Purge everything from specified zones
- Apply operations to specific zones or all zones
- Support for both API Token and API Key authentication methods

## Building from Source

### Prerequisites

- Go 1.18 or higher
- Git (for cloning the repository)

### Build Steps

1. Clone the repository (if you haven't already):
   ```bash
   git clone https://github.com/erfianugrah/cache-purge-go
   cd cache-purge-go
   ```

2. Install dependencies:
   ```bash
   go mod init cfpurge
   go mod tidy
   ```

3. Build the binary:
   ```bash
   go build -o dist/cfpurge
   ```

4. (Optional) Install to your Go bin directory:
   ```bash
   go install
   ```

The built binary will be in the `dist` directory. You can move it to any location in your PATH, for example:

```bash
sudo mv dist/cfpurge /usr/local/bin/
```

### Cross-compilation

To build for different platforms:

```bash
# For Linux
GOOS=linux GOARCH=amd64 go build -o dist/cfpurge-linux-amd64

# For macOS
GOOS=darwin GOARCH=amd64 go build -o dist/cfpurge-darwin-amd64

# For Windows
GOOS=windows GOARCH=amd64 go build -o dist/cfpurge-windows-amd64.exe
```

## Configuration

The tool supports authentication via either API Token or API Key + Email combination. You can provide these credentials in two ways:

### Environment Variables

```bash
# Using API Token (recommended)
export CLOUDFLARE_API_TOKEN="your-api-token"

# Or using API Key + Email
export CLOUDFLARE_API_KEY="your-api-key"
export CLOUDFLARE_EMAIL="your-email@example.com"

# Optional Account ID for filtering zones
export CLOUDFLARE_ACCOUNT_ID="your-account-id"
```

### Command Line Flags

```bash
# Using API Token
cfpurge -token="your-api-token" ...

# Or using API Key + Email
cfpurge -key="your-api-key" -email="your-email@example.com" ...
```

## Usage

### List Available Zones

```bash
cfpurge list
```

### Purge Cache Operations

#### Purge Everything from a Zone

```bash
cfpurge purge -everything example.com
```

#### Purge by Hosts

```bash
cfpurge purge -hosts="api.example.com,www.example.com"
```

#### Purge by URLs

```bash
cfpurge purge -urls="https://example.com/page1,https://example.com/page2"
```

#### Purge by Tags (Enterprise Only)

```bash
cfpurge purge -tags="tag1,tag2"
```

#### Purge Across Multiple Zones

```bash
cfpurge purge -hosts="api.example.com" example.com example.org
```

#### Purge from All Zones

```bash
cfpurge purge -all -hosts="api.example.com"
```

### Additional Options

- `-quiet`: Suppress success messages
- `-account`: Specify Cloudflare account ID

## Examples

1. List all zones:
```bash
cfpurge list
```

2. Purge everything from a specific zone:
```bash
cfpurge purge -everything example.com
```

3. Purge specific hosts across all zones:
```bash
cfpurge purge -all -hosts="api.example.com,www.example.com"
```

4. Purge URLs from specific zones with quiet output:
```bash
cfpurge purge -quiet -urls="https://example.com/page1" example.com
```

## Error Handling

- The tool will display clear error messages when operations fail
- Exit codes:
  - 0: Success
  - 1: Error (authentication, API errors, no matching zones, etc.)
- A summary of successful and failed operations is displayed at the end

## Dependencies

- [cloudflare-go](https://github.com/cloudflare/cloudflare-go): Official Cloudflare Go SDK

## License

MIT License - see LICENSE file for details
