#!/bin/bash
#
# n8n CLI installer
#
# Usage:
#   gh api repos/enthus-appdev/n8n-cli/contents/install.sh -q '.content' | base64 -d | bash
#
# Or clone and run:
#   gh repo clone enthus-appdev/n8n-cli && cd n8n-cli && ./install.sh
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
REPO="enthus-appdev/n8n-cli"
BINARY_NAME="n8nctl"
GO_MODULE="github.com/${REPO}/cmd/n8nctl"

echo -e "${BLUE}${BOLD}n8n CLI Installer${NC}"
echo

# Check for Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}Go is not installed.${NC}"
    echo "Please install Go first: https://go.dev/doc/install"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo -e "${GREEN}✓${NC} Go ${GO_VERSION} detected"

# Install using go install
echo
echo -e "${BLUE}Installing n8n-cli...${NC}"

if go install "${GO_MODULE}@latest"; then
    echo -e "${GREEN}✓${NC} n8n-cli installed successfully"
else
    echo -e "${YELLOW}go install failed, trying to build from source...${NC}"

    # Clone and build
    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"

    echo -e "${BLUE}Cloning repository...${NC}"
    git clone "https://github.com/${REPO}.git" --depth 1
    cd n8n-cli

    echo -e "${BLUE}Building...${NC}"
    go build -o "${BINARY_NAME}" ./cmd/n8nctl

    # Try to install to common locations
    INSTALLED=false
    for DIR in "${GOPATH:-$HOME/go}/bin" "$HOME/.local/bin" "/usr/local/bin"; do
        if [ -d "$DIR" ] && [ -w "$DIR" ]; then
            cp "${BINARY_NAME}" "$DIR/"
            echo -e "${GREEN}✓${NC} Installed to $DIR/${BINARY_NAME}"
            INSTALLED=true
            break
        fi
    done

    if [ "$INSTALLED" = false ]; then
        # Try with sudo for /usr/local/bin
        if [ -d "/usr/local/bin" ]; then
            sudo cp "${BINARY_NAME}" /usr/local/bin/
            echo -e "${GREEN}✓${NC} Installed to /usr/local/bin/${BINARY_NAME}"
        else
            echo -e "${RED}Could not find a writable install location${NC}"
            echo "Please copy the binary manually from: $(pwd)/${BINARY_NAME}"
            exit 1
        fi
    fi

    # Cleanup
    cd
    rm -rf "$TEMP_DIR"
fi

# Verify installation
echo
if command -v "${BINARY_NAME}" &> /dev/null; then
    VERSION=$("${BINARY_NAME}" --version 2>/dev/null || echo "unknown")
    echo -e "${GREEN}${BOLD}✓ n8n-cli is ready!${NC}"
    echo
    echo "Get started:"
    echo -e "  ${BLUE}n8nctl config init${NC}           # Configure your n8n instance"
    echo -e "  ${BLUE}n8nctl workflow list${NC}         # List workflows"
    echo -e "  ${BLUE}n8nctl workflow pull <id> -r${NC} # Pull workflow with sub-workflows"
else
    echo -e "${YELLOW}Warning: ${BINARY_NAME} command not found in PATH${NC}"
    echo "You may need to add ~/go/bin to your PATH:"
    echo '  export PATH="$PATH:$HOME/go/bin"'
fi
