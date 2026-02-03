#!/bin/bash
#
# shhh install script
# https://github.com/cychiuae/shhh
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/cychiuae/shhh/main/install.sh | bash
#   or
#   ./install.sh
#
# Options:
#   INSTALL_DIR  - Installation directory (default: /usr/local/bin or ~/.local/bin)
#   VERSION      - Version to install (default: latest)

set -e

REPO="cychiuae/shhh"
BINARY_NAME="shhh"
INSTALL_DIR="${INSTALL_DIR:-}"
VERSION="${VERSION:-latest}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Detect OS
detect_os() {
    local os
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$os" in
        linux*)  echo "linux" ;;
        darwin*) echo "darwin" ;;
        mingw*|msys*|cygwin*) echo "windows" ;;
        *)       error "Unsupported operating system: $os" ;;
    esac
}

# Detect architecture
detect_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)  echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *)             error "Unsupported architecture: $arch" ;;
    esac
}

# Determine install directory
get_install_dir() {
    if [ -n "$INSTALL_DIR" ]; then
        echo "$INSTALL_DIR"
        return
    fi

    # Try /usr/local/bin first (requires sudo)
    if [ -w "/usr/local/bin" ]; then
        echo "/usr/local/bin"
        return
    fi

    # Fall back to ~/.local/bin
    local local_bin="$HOME/.local/bin"
    mkdir -p "$local_bin"
    echo "$local_bin"
}

# Check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Get the latest version from GitHub
get_latest_version() {
    if command_exists curl; then
        curl -sS "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/' || echo ""
    elif command_exists wget; then
        wget -qO- "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/' || echo ""
    fi
}

# Download a file
download() {
    local url="$1"
    local output="$2"

    if command_exists curl; then
        curl -fsSL "$url" -o "$output"
    elif command_exists wget; then
        wget -q "$url" -O "$output"
    else
        error "Neither curl nor wget found. Please install one of them."
    fi
}

# Try to install from GitHub releases
install_from_release() {
    local os="$1"
    local arch="$2"
    local install_dir="$3"
    local version="$4"

    if [ "$version" = "latest" ]; then
        version=$(get_latest_version)
        if [ -z "$version" ]; then
            return 1
        fi
    fi

    info "Downloading shhh $version for $os/$arch..."

    local ext=""
    [ "$os" = "windows" ] && ext=".exe"

    local download_url="https://github.com/$REPO/releases/download/$version/${BINARY_NAME}-${os}-${arch}${ext}"
    local tmp_file
    tmp_file=$(mktemp)

    if download "$download_url" "$tmp_file" 2>/dev/null; then
        chmod +x "$tmp_file"

        # Check if we need sudo
        if [ -w "$install_dir" ]; then
            mv "$tmp_file" "$install_dir/$BINARY_NAME"
        else
            info "Installing to $install_dir requires elevated privileges..."
            sudo mv "$tmp_file" "$install_dir/$BINARY_NAME"
        fi
        return 0
    else
        rm -f "$tmp_file"
        return 1
    fi
}

# Install from source using Go
install_from_source() {
    local install_dir="$1"

    if ! command_exists go; then
        error "Go is not installed. Please install Go 1.21+ from https://go.dev/dl/"
    fi

    # Check Go version
    local go_version
    go_version=$(go version | sed -E 's/.*go([0-9]+\.[0-9]+).*/\1/')
    local major minor
    major=$(echo "$go_version" | cut -d. -f1)
    minor=$(echo "$go_version" | cut -d. -f2)

    if [ "$major" -lt 1 ] || { [ "$major" -eq 1 ] && [ "$minor" -lt 21 ]; }; then
        error "Go 1.21+ is required. Found: go$go_version"
    fi

    info "Building shhh from source..."

    # Create temp directory
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT

    # Clone the repository
    info "Cloning repository..."
    if command_exists git; then
        git clone --depth 1 "https://github.com/$REPO.git" "$tmp_dir/shhh" 2>/dev/null
    else
        error "Git is not installed. Please install git."
    fi

    # Build
    cd "$tmp_dir/shhh"
    info "Compiling..."

    local version
    version=$(git describe --tags --always --dirty 2>/dev/null || echo "0.1.0")
    local build_time
    build_time=$(date -u '+%Y-%m-%dT%H:%M:%SZ')

    go build -ldflags "-X github.com/cychiuae/shhh/cmd.Version=$version -X github.com/cychiuae/shhh/cmd.BuildTime=$build_time" -o "$BINARY_NAME" .

    # Install
    if [ -w "$install_dir" ]; then
        mv "$BINARY_NAME" "$install_dir/"
    else
        info "Installing to $install_dir requires elevated privileges..."
        sudo mv "$BINARY_NAME" "$install_dir/"
    fi
}

# Main installation logic
main() {
    echo ""
    echo "  ___| |     |     |    "
    echo " \___ | |__  | |__ | |__"
    echo "       _|   \|   \|   \ "
    echo " _____/  _| _| _| _| _| "
    echo ""
    echo "GitOps-friendly secret management"
    echo ""

    local os arch install_dir
    os=$(detect_os)
    arch=$(detect_arch)
    install_dir=$(get_install_dir)

    info "Detected: $os/$arch"
    info "Install directory: $install_dir"

    # Try downloading pre-built binary first
    if install_from_release "$os" "$arch" "$install_dir" "$VERSION"; then
        success "Successfully installed shhh to $install_dir/$BINARY_NAME"
    else
        warn "Pre-built binary not available. Building from source..."
        install_from_source "$install_dir"
        success "Successfully built and installed shhh to $install_dir/$BINARY_NAME"
    fi

    # Check if install directory is in PATH
    if ! echo "$PATH" | grep -q "$install_dir"; then
        echo ""
        warn "The install directory is not in your PATH."
        echo ""
        echo "Add it to your shell configuration:"
        echo ""
        echo "  # For bash (add to ~/.bashrc):"
        echo "  export PATH=\"$install_dir:\$PATH\""
        echo ""
        echo "  # For zsh (add to ~/.zshrc):"
        echo "  export PATH=\"$install_dir:\$PATH\""
        echo ""
        echo "  # For fish (add to ~/.config/fish/config.fish):"
        echo "  fish_add_path $install_dir"
        echo ""
    fi

    # Verify installation
    if [ -x "$install_dir/$BINARY_NAME" ]; then
        echo ""
        info "Verifying installation..."
        "$install_dir/$BINARY_NAME" --version 2>/dev/null || true
        echo ""
        success "Installation complete! Run 'shhh --help' to get started."
    fi
}

main "$@"
