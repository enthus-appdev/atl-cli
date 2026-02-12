#!/bin/bash
#
# Atlassian CLI (atl) installer
#
# Usage:
#   gh api repos/enthus-appdev/atl-cli/contents/install.sh -q '.content' | base64 -d | bash
#
# Or clone and run:
#   gh repo clone enthus-appdev/atl-cli && cd atl-cli && ./install.sh
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Configuration
REPO="enthus-appdev/atl-cli"
BINARY_NAME="atl"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        darwin) OS="darwin" ;;
        linux) OS="linux" ;;
        mingw*|msys*|cygwin*) OS="windows" ;;
        *)
            echo -e "${RED}Unsupported operating system: $OS${NC}"
            exit 1
            ;;
    esac

    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        armv7l) ARCH="arm" ;;
        *)
            echo -e "${RED}Unsupported architecture: $ARCH${NC}"
            exit 1
            ;;
    esac

    PLATFORM="${OS}_${ARCH}"
}

# Check for required tools
check_dependencies() {
    local missing=()

    for cmd in curl tar; do
        if ! command -v "$cmd" &> /dev/null; then
            missing+=("$cmd")
        fi
    done

    if [ ${#missing[@]} -ne 0 ]; then
        echo -e "${RED}Missing required tools: ${missing[*]}${NC}"
        echo "Please install them and try again."
        exit 1
    fi
}

# Get latest release version
get_latest_version() {
    local version
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

    if [ -z "$version" ]; then
        echo -e "${YELLOW}Could not fetch latest version, using 'latest'${NC}"
        version="latest"
    fi

    echo "$version"
}

# Download and install
install_binary() {
    local version="$1"
    local tmp_dir
    tmp_dir=$(mktemp -d)

    # Construct download URL
    local download_url
    if [ "$version" = "latest" ]; then
        download_url="https://github.com/${REPO}/releases/latest/download/${BINARY_NAME}_${PLATFORM}.tar.gz"
    else
        download_url="https://github.com/${REPO}/releases/download/${version}/${BINARY_NAME}_${PLATFORM}.tar.gz"
    fi

    echo -e "${BLUE}Downloading ${BINARY_NAME} ${version} for ${PLATFORM}...${NC}"

    if ! curl -fsSL "$download_url" -o "${tmp_dir}/${BINARY_NAME}.tar.gz" 2>/dev/null; then
        echo -e "${YELLOW}Pre-built binary not found. Building from source...${NC}"
        install_from_source
        return
    fi

    echo "Extracting..."
    tar -xzf "${tmp_dir}/${BINARY_NAME}.tar.gz" -C "$tmp_dir"

    echo "Installing to ${INSTALL_DIR}..."
    if [ -w "$INSTALL_DIR" ]; then
        mv "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        echo -e "${YELLOW}Need sudo to install to ${INSTALL_DIR}${NC}"
        sudo mv "${tmp_dir}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    rm -rf "$tmp_dir"
}

# Build and install from source
install_from_source() {
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Go is not installed. Please install Go 1.21+ or download a pre-built release.${NC}"
        echo "Install Go: https://go.dev/dl/"
        exit 1
    fi

    echo -e "${BLUE}Building from source...${NC}"

    local tmp_dir
    tmp_dir=$(mktemp -d)

    # Try gh clone first (works for private repos), fall back to SSH
    if command -v gh &> /dev/null; then
        gh repo clone "${REPO}" "$tmp_dir" -- --depth 1 2>/dev/null || {
            echo -e "${RED}Failed to clone repository${NC}"
            exit 1
        }
    else
        git clone --depth 1 "https://github.com/${REPO}.git" "$tmp_dir" 2>/dev/null || {
            echo -e "${RED}Failed to clone repository. Check network connectivity or repository visibility.${NC}"
            exit 1
        }
    fi

    cd "$tmp_dir"
    go build -o "${BINARY_NAME}" ./cmd/atl

    echo "Installing to ${INSTALL_DIR}..."
    if [ -w "$INSTALL_DIR" ]; then
        mv "${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        echo -e "${YELLOW}Need sudo to install to ${INSTALL_DIR}${NC}"
        sudo mv "${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
    cd - > /dev/null
    rm -rf "$tmp_dir"
}

# Verify installation
verify_installation() {
    if command -v "$BINARY_NAME" &> /dev/null; then
        echo ""
        echo -e "${GREEN}${BOLD}Successfully installed ${BINARY_NAME}!${NC}"
        echo ""
        "$BINARY_NAME" --version
        echo ""
    else
        echo -e "${YELLOW}Installation complete, but ${BINARY_NAME} is not in PATH.${NC}"
        echo "Add ${INSTALL_DIR} to your PATH, or run:"
        echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
        echo ""
    fi
}

# Print next steps
print_next_steps() {
    echo -e "${BOLD}Next steps:${NC}"
    echo ""
    echo "  1. Set up OAuth authentication:"
    echo ""
    echo -e "     ${BLUE}atl auth setup${NC}"
    echo ""
    echo "     This will guide you through creating an Atlassian OAuth app"
    echo "     and storing the credentials securely."
    echo ""
    echo "  2. Log in to your Atlassian account:"
    echo ""
    echo -e "     ${BLUE}atl auth login${NC}"
    echo ""
    echo "  3. Start using the CLI:"
    echo ""
    echo -e "     ${BLUE}atl issue list --assignee @me${NC}"
    echo -e "     ${BLUE}atl confluence space list${NC}"
    echo ""
    echo -e "For help, run: ${BLUE}atl --help${NC}"
    echo ""
}

# Main
main() {
    echo ""
    echo -e "${BOLD}Atlassian CLI Installer${NC}"
    echo ""

    check_dependencies
    detect_platform

    echo -e "Platform: ${BLUE}${PLATFORM}${NC}"

    VERSION=$(get_latest_version)
    echo -e "Version:  ${BLUE}${VERSION}${NC}"
    echo ""

    install_binary "$VERSION"
    verify_installation
    print_next_steps
}

main "$@"
