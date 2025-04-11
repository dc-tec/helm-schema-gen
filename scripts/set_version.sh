#!/bin/sh

## This script performs two actions:
## 1. Updates version references in plugin.yaml
## 2. Updates the version used in the installation

# Get version from first argument or use default
VERSION=${1:-"v1.1.6"}

# If VERSION doesn't start with 'v', add it
case "$VERSION" in
  v*) ;;
  *) VERSION="v$VERSION" ;;
esac

# Update plugin.yaml version
if command -v yq >/dev/null 2>&1; then
  # Test if yq supports the 'write' command (older versions)
  if yq --help | grep -q write; then
    yq write -i plugin.yaml version "$VERSION"
  else
    # Newer versions of yq use different syntax
    yq eval ".version = \"$VERSION\"" -i plugin.yaml
  fi
else
  echo "Warning: yq not found. Cannot update plugin.yaml version automatically."
  echo "Please update plugin.yaml version to $VERSION manually."
fi

# Self-update: replace the hardcoded version in this file
# Get OS for proper sed command
OS=$(uname)
if [ "$OS" = "Darwin" ]; then
  # macOS requires an empty string with -i
  sed -i '' "s/VERSION=\${1:-\"[^\"]*\"}/VERSION=\${1:-\"$VERSION\"}/" "$0"
else
  # Linux version
  sed -i "s/VERSION=\${1:-\"[^\"]*\"}/VERSION=\${1:-\"$VERSION\"}/" "$0"
fi
