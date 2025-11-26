#!/bin/bash

# Build script for WiiUDownloader API server
# This script builds the Docker container for the API server

set -e

echo "Building WiiUDownloader API Docker container..."

# Generate title database if needed
if [ ! -f "db.go" ]; then
    echo "Generating title database..."
    python3 grabTitles.py
fi

# Build Docker image
echo "Building Docker image..."
docker build -f Dockerfile.api -t wiiu-api:latest .

echo "Build complete!"
echo ""
echo "To run the API server:"
echo "  docker run -p 11235:11235 -v \$(pwd)/downloads:/downloads wiiu-api:latest"
echo ""
echo "Or use docker-compose:"
echo "  docker-compose up -d"
echo ""
echo "API will be available at: http://localhost:11235"
echo "Health check: http://localhost:11235/health"
echo "API docs: http://localhost:11235/api/docs (when implemented)"
