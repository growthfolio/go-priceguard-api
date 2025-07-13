#!/bin/bash

# PriceGuard API Development Start Script
set -e

echo "🚀 Starting PriceGuard API in development mode..."

# Create tmp directory if it doesn't exist
echo "📁 Creating tmp directory..."
mkdir -p /app/tmp

# Set correct permissions
chmod 755 /app/tmp

echo "🔨 Building initial binary..."
# Build the initial binary with buildvcs disabled
CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -o /app/tmp/main /app/cmd/server

# Verify the binary was created
if [ ! -f "/app/tmp/main" ]; then
    echo "❌ Error: Binary /app/tmp/main was not created!"
    exit 1
fi

echo "✅ Initial binary built successfully"
echo "🔄 Starting air for hot reload..."

# Start air with the provided arguments
exec "$@"
