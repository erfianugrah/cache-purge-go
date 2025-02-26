a#!/bin/bash

# This script updates all import paths in Go files
# from GitHub paths to local module paths

# Set the module name
MODULE_NAME="cfpurge"

# Find all Go files in the project
GO_FILES=$(find . -type f -name "*.go")

# Replace GitHub import paths with local module paths
for file in $GO_FILES; do
  echo "Processing $file..."
  
  # Replace import paths
  sed -i 's|github.com/erfianugrah/cache-purge-go|cfpurge|g' "$file"
  sed -i 's|cf-purge|cfpurge|g' "$file"
done

echo "All import paths updated to use '$MODULE_NAME'"
