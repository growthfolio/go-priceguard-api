#!/bin/bash

# PriceGuard API Development Start Script
set -e

echo "🚀 Starting PriceGuard API Development Environment..."

# Create tmp directory if it doesn't exist
echo "📁 Creating tmp directory..."
mkdir -p /app/tmp
chmod 755 /app/tmp

# Wait for dependencies
echo "⏳ Waiting for dependencies..."
sleep 5

echo "📦 Building initial binary..."

# Build the application first to ensure it works
if CGO_ENABLED=0 go build -buildvcs=false -o /app/tmp/main ./cmd/server/; then
    echo "✅ Initial build successful!"
else
    echo "❌ Initial build failed. Attempting recovery..."
    
    # Try to fix common issues
    go mod tidy
    go mod download
    
    # Try building again
    if CGO_ENABLED=0 go build -buildvcs=false -o /app/tmp/main ./cmd/server/; then
        echo "✅ Build successful after recovery!"
    else
        echo "❌ Build failed permanently. Starting air anyway (will retry)..."
    fi
fi

echo "🌪️ Starting air for hot reload..."

# Execute the command passed to the script
exec "$@"
