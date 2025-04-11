#!/bin/sh

## This script performs two actions:
## 1. Updates version references in plugin.yaml
## 2. Updates the version used in the installation

# Get version from first argument or use default
VERSION=${1:-"1.0.0"}

# Update plugin.yaml version
if command -v yq >/dev/null 2>&1; then
  yq write -i plugin.yaml version "$VERSION"
else
  echo "Warning: yq not found. Cannot update plugin.yaml version automatically."
  echo "Please update plugin.yaml version to $VERSION manually."
fi

# Run the installer with the specified version
./scripts/install.sh "$VERSION"

# Self-update: replace the hardcoded version in this file
# Get OS for proper sed command
OS=$(uname)
if [ "$OS" = "Darwin" ]; then
  # macOS requires an empty string with -i
  sed -i '' "s/VERSION=\${1:-\"[0-9]*\.[0-9]*\.[0-9]*\"}/VERSION=\${1:-\"$VERSION\"}/" "$0"
else
  # Linux version
  sed -i "s/VERSION=\${1:-\"[0-9]*\.[0-9]*\.[0-9]*\"}/VERSION=\${1:-\"$VERSION\"}/" "$0"
fi
