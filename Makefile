BINARY_NAME=cfpurge
DIST_DIR=dist
MAIN_PACKAGE=.
GO_FILES=$(shell find . -type f -name "*.go")
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}"

.PHONY: all build clean test install uninstall fmt lint vet help cross-build

all: clean build

build: $(GO_FILES) ## Build the binary
	@echo "Building ${BINARY_NAME}..."
	@mkdir -p $(DIST_DIR)
	@go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

clean: ## Remove build artifacts
	@echo "Cleaning..."
	@rm -rf $(DIST_DIR)

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@mkdir -p $(DIST_DIR)
	@go test -v -coverprofile=$(DIST_DIR)/coverage.out ./...
	@go tool cover -html=$(DIST_DIR)/coverage.out -o $(DIST_DIR)/coverage.html
	@echo "Coverage report generated at $(DIST_DIR)/coverage.html"

install: build ## Install the binary
	@echo "Installing..."
	@cp $(DIST_DIR)/$(BINARY_NAME) /usr/local/bin/

uninstall: ## Remove the installed binary
	@echo "Uninstalling..."
	@rm -f /usr/local/bin/$(BINARY_NAME)

fmt: ## Format the code
	@echo "Formatting code..."
	@go fmt ./...

lint: ## Lint the code
	@echo "Linting code..."
	@if [ -x "$(command -v golangci-lint)" ]; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found. Please install it: https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

vet: ## Vet the code
	@echo "Vetting code..."
	@go vet ./...

cross-build: ## Build for multiple platforms
	@echo "Building releases..."
	@mkdir -p $(DIST_DIR)
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PACKAGE)
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	@GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-arm64.exe $(MAIN_PACKAGE)
	@echo "Done building releases."
	@ls -lh $(DIST_DIR)

help: ## Show this help
	@echo "Makefile targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Default target
.DEFAULT_GOAL := help
