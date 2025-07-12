#!/bin/bash

# Script para atualizar todos os imports do módulo

OLD_MODULE="github.com/growthfolio/go-priceguard-api"
NEW_MODULE="github.com/growthfolio/go-priceguard-api"

echo "Updating module imports from $OLD_MODULE to $NEW_MODULE"

# Find all .go files and update imports
find . -type f -name "*.go" -not -path "./vendor/*" | while read -r file; do
    if grep -q "$OLD_MODULE" "$file"; then
        echo "Updating: $file"
        sed -i "s|$OLD_MODULE|$NEW_MODULE|g" "$file"
    fi
done

echo "Running go mod tidy..."
go mod tidy

echo "Module update complete!"
