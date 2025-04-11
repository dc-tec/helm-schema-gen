#!/bin/sh
set -e

# Simple installer for helm-schema-gen
# Replaces the deprecated godownloader script

GITHUB_REPO="dc-tec/helm-schema-gen"
PROJECT_NAME="helm-schema-gen"
BINDIR=${BINDIR:-"$(pwd)/bin"}
TAG=$1

# Create bin directory if it doesn't exist
mkdir -p "$BINDIR"

# Log functions
log_info() {
  echo "[INFO] $1"
}

log_err() {
  echo "[ERROR] $1" >&2
}

# Check if command exists
is_command() {
  command -v "$1" >/dev/null
}

# Detect OS and architecture
detect_os() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$os" in
    cygwin*|mingw*|msys*) os="windows" ;;
  esac
  echo "$os"
}

detect_arch() {
  arch=$(uname -m)
  case $arch in
    x86_64) arch="amd64" ;;
    x86|i686|i386) arch="386" ;;
    aarch64) arch="arm64" ;;
    armv5*) arch="armv5" ;;
    armv6*) arch="armv6" ;;
    armv7*) arch="armv7" ;;
  esac
  echo $arch
}

# Download file using curl or wget
download_file() {
  url=$1
  output=$2
  
  if is_command curl; then
    curl -sSL -o "$output" "$url" || return 1
  elif is_command wget; then
    wget -q -O "$output" "$url" || return 1
  else
    log_err "Neither curl nor wget found. Please install one of them."
    exit 1
  fi
}

# Get latest release if no tag specified
get_latest_release() {
  repo=$1
  
  if is_command curl; then
    # Use the -f flag to fail silently on error, and follow redirects with -L
    latest_tag=$(curl -sSLf "https://api.github.com/repos/$repo/releases/latest" | 
      grep -o '"tag_name": "[^"]*' | 
      sed 's/"tag_name": "//') || return 1
    
    # If the GitHub API request fails or returns empty, use a fallback minimum version
    if [ -z "$latest_tag" ]; then
      log_info "Failed to get latest release via API, using fallback minimum version v1.1.6"
      echo "v1.1.6"
      return 0
    fi
    
    echo "$latest_tag"
  elif is_command wget; then
    latest_tag=$(wget -q -O- "https://api.github.com/repos/$repo/releases/latest" |
      grep -o '"tag_name": "[^"]*' | 
      sed 's/"tag_name": "//') || return 1
    
    # If the GitHub API request fails or returns empty, use a fallback minimum version
    if [ -z "$latest_tag" ]; then
      log_info "Failed to get latest release via API, using fallback minimum version v1.1.6"
      echo "v1.1.6"
      return 0
    fi
    
    echo "$latest_tag"
  else
    log_err "Neither curl nor wget found. Please install one of them."
    exit 1
  fi
}

# Verify SHA-256 checksum (if available)
verify_checksum() {
  file=$1
  expected=$2
  
  if [ -z "$expected" ]; then
    log_info "Skipping checksum verification - no checksum found"
    return 0
  fi
  
  if is_command sha256sum; then
    actual=$(sha256sum "$file" | cut -d ' ' -f 1)
  elif is_command shasum; then
    actual=$(shasum -a 256 "$file" | cut -d ' ' -f 1)
  elif is_command openssl; then
    actual=$(openssl dgst -sha256 "$file" | sed 's/^.* //')
  else
    log_err "No checksum tool found. Cannot verify download."
    return 0  # Skip verification as a fallback
  fi
  
  if [ "$actual" != "$expected" ]; then
    log_err "Checksum verification failed! Expected: $expected, got: $actual"
    return 1
  fi
  
  log_info "Checksum verification passed"
  return 0
}

# Main installation logic
main() {
  # Detect OS and architecture
  OS=$(detect_os)
  ARCH=$(detect_arch)
  
  # Get latest release if tag not specified
  if [ -z "$TAG" ]; then
    log_info "No version specified, fetching latest release..."
    TAG=$(get_latest_release "$GITHUB_REPO")
    if [ -z "$TAG" ]; then
      # If get_latest_release still fails, use a minimum known good version
      log_err "Failed to get latest release, falling back to v1.1.6"
      TAG="v1.1.6"
    fi
  fi
  
  # Strip 'v' prefix from tag if present
  VERSION=${TAG#v}
  
  log_info "Installing $PROJECT_NAME $VERSION for $OS/$ARCH"
  
  # Determine file format
  FORMAT="tar.gz"
  if [ "$OS" = "windows" ]; then
    FORMAT="zip"
  fi
  
  # Setup URLs and filenames
  BINARY_NAME="$PROJECT_NAME"
  RELEASE_NAME="${PROJECT_NAME}_${VERSION}_${OS}_${ARCH}"
  TARBALL="${RELEASE_NAME}.${FORMAT}"
  TARBALL_URL="https://github.com/${GITHUB_REPO}/releases/download/${TAG}/${TARBALL}"
  CHECKSUM_URL="https://github.com/${GITHUB_REPO}/releases/download/${TAG}/${PROJECT_NAME}_${VERSION}_checksums.txt"
  
  # Create temporary directory
  TMP_DIR=$(mktemp -d)
  
  # Download files
  log_info "Downloading $TARBALL_URL"
  download_file "$TARBALL_URL" "$TMP_DIR/$TARBALL" || {
    log_err "Failed to download $TARBALL_URL"
    rm -rf "$TMP_DIR"
    exit 1
  }
  
  # Try to download checksums, but continue even if it fails
  log_info "Downloading checksums"
  EXPECTED_SUM=""
  if download_file "$CHECKSUM_URL" "$TMP_DIR/checksums.txt" 2>/dev/null; then
    # Look for the checksum in different formats
    EXPECTED_SUM=$(grep -E "(^| )${TARBALL}($| )" "$TMP_DIR/checksums.txt" | awk '{print $1}')
    if [ -z "$EXPECTED_SUM" ]; then
      # Try alternative name format
      local BASENAME=$(basename "$TARBALL")
      EXPECTED_SUM=$(grep -E "(^| )${BASENAME}($| )" "$TMP_DIR/checksums.txt" | awk '{print $1}')
    fi
  else
    log_info "No checksum file found - continuing without verification"
  fi
  
  # Verify checksum if we found one
  if [ -n "$EXPECTED_SUM" ]; then
    verify_checksum "$TMP_DIR/$TARBALL" "$EXPECTED_SUM" || {
      rm -rf "$TMP_DIR"
      exit 1
    }
  fi
  
  # Create bin directory if it doesn't exist
  if [ ! -d "$BINDIR" ]; then
    mkdir -p "$BINDIR"
  fi
  
  # Extract archive
  log_info "Extracting $TARBALL"
  cd "$TMP_DIR"
  
  case "$FORMAT" in
    "tar.gz")
      if ! tar -xzf "$TARBALL" 2>/dev/null; then
        # If that fails, try gunzip and tar separately
        log_info "Trying alternative extraction method"
        if is_command gunzip; then
          gunzip -c "$TARBALL" | tar -xf - || {
            log_err "Failed to extract archive"
            rm -rf "$TMP_DIR"
            exit 1
          }
        else
          log_err "Failed to extract archive and gunzip not available"
          rm -rf "$TMP_DIR"
          exit 1
        fi
      fi
      ;;
    "zip")
      if is_command unzip; then
        unzip -q "$TARBALL" || {
          log_err "Failed to extract archive"
          rm -rf "$TMP_DIR"
          exit 1
        }
      else
        log_err "unzip command not found. Please install unzip."
        rm -rf "$TMP_DIR"
        exit 1
      fi
      ;;
  esac
  
  # Look for the binary - it might be in a subdirectory or directly in the archive
  if [ ! -f "$BINARY_NAME" ]; then
    # Try to find it
    BINARY_PATH=$(find . -name "$BINARY_NAME" -type f | head -n 1)
    if [ -z "$BINARY_PATH" ]; then
      if [ "$OS" = "windows" ]; then
        BINARY_PATH=$(find . -name "$BINARY_NAME.exe" -type f | head -n 1)
      fi
    fi
    
    if [ -z "$BINARY_PATH" ]; then
      log_err "Binary not found in extracted archive"
      ls -la
      rm -rf "$TMP_DIR"
      exit 1
    fi
    
    BINARY_NAME="$BINARY_PATH"
  fi
  
  # Install binary
  log_info "Installing $BINARY_NAME to $BINDIR"
  if is_command install; then
    install -m 755 "$BINARY_NAME" "$BINDIR/" || {
      log_err "Failed to install $BINARY_NAME"
      rm -rf "$TMP_DIR"
      exit 1
    }
  else
    # Fallback if 'install' command is not available
    cp "$BINARY_NAME" "$BINDIR/" && chmod 755 "$BINDIR/$(basename "$BINARY_NAME")" || {
      log_err "Failed to install $BINARY_NAME"
      rm -rf "$TMP_DIR"
      exit 1
    }
  fi
  
  # Cleanup
  cd - >/dev/null
  rm -rf "$TMP_DIR"
  
  log_info "Successfully installed $PROJECT_NAME $VERSION to $BINDIR/$(basename "$BINARY_NAME")"
}

# Run installer
main
