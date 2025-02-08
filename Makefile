BINARY_NAME=cfpurge
DIST_DIR=dist

.PHONY: all build clean test install uninstall fmt lint

all: clean build

build:
	@echo "Building..."
	@mkdir -p $(DIST_DIR)
	@go build -o $(DIST_DIR)/$(BINARY_NAME)

clean:
	@echo "Cleaning..."
	@rm -rf $(DIST_DIR)

test:
	@echo "Running tests..."
	@go test -v ./...

install: build
	@echo "Installing..."
	@cp $(DIST_DIR)/$(BINARY_NAME) /usr/local/bin/

uninstall:
	@echo "Uninstalling..."
	@rm -f /usr/local/bin/$(BINARY_NAME)

fmt:
	@echo "Formatting code..."
	@go fmt ./...

lint:
	@echo "Linting code..."
	@golint ./...

release:
	@echo "Building releases..."
	@mkdir -p $(DIST_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64
	@GOOS=darwin GOARCH=amd64 go build -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64
	@GOOS=windows GOARCH=amd64 go build -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe
