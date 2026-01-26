#!/usr/bin/env bash

set -e

REPO="REDFOX1899/ask-sh"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="x"

echo "Installing x - Natural Language Shell Command Executor..."
echo ""

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

echo "Detected: $OS/$ARCH"

# Check if Go is installed for building from source
if command -v go &> /dev/null; then
    echo "Go detected, building from source..."

    # Create temp directory
    TEMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TEMP_DIR"' EXIT

    # Clone and build
    git clone --depth 1 "https://github.com/$REPO.git" "$TEMP_DIR/ask-sh"
    cd "$TEMP_DIR/ask-sh"
    go build -o "$BINARY_NAME" ./cmd/x

    # Install
    if [ -w "$INSTALL_DIR" ]; then
        cp "$BINARY_NAME" "$INSTALL_DIR/"
    else
        echo "Installing to $INSTALL_DIR (requires sudo)..."
        sudo cp "$BINARY_NAME" "$INSTALL_DIR/"
    fi
else
    echo "Go not found. Please install Go first: https://go.dev/dl/"
    echo ""
    echo "Or install Go and run:"
    echo "  curl -sSL https://raw.githubusercontent.com/$REPO/master/install.sh | bash"
    exit 1
fi

echo ""
echo "✓ x installed successfully to $INSTALL_DIR/$BINARY_NAME"
echo ""
echo "Set your API key (choose one):"
echo "  export OPENAI_API_KEY=\"your-key\""
echo "  export ANTHROPIC_API_KEY=\"your-key\""
echo "  export GEMINI_API_KEY=\"your-key\""
echo "  export OLLAMA_MODEL=\"llama3.2\""
echo ""
echo "Usage: x <natural language instruction>"
echo ""
echo "Examples:"
echo "  x list all git branches"
echo "  x find files modified in last 7 days"
echo "  x show disk usage"
