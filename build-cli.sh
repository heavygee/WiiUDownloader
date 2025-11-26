#!/bin/bash

# Build script for WiiUDownloader CLI version
# This script builds the CLI version that can be used without GUI

set -e

echo "Building WiiUDownloader CLI..."

# Generate the title database if it doesn't exist
if [ ! -f "db.go" ]; then
    echo "Generating title database..."
    python3 grabTitles.py
fi

# Build the CLI binary
echo "Building CLI binary..."
go build -o wiiu-cli ./cmd/wiiu-cli

echo "Build complete! Binary: wiiu-cli"
echo ""
echo "Usage examples:"
echo "  ./wiiu-cli -list -category game                    # List all games"
echo "  ./wiiu-cli -search \"Mario\" -category game         # Search for Mario games"
echo "  ./wiiu-cli -title 00050000101C9500 -output ./downloads  # Download a specific title"
echo "  ./wiiu-cli -title 00050000101C9500 -output ./downloads -decrypt  # Download and decrypt"
