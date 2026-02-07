#!/usr/bin/env bash

set -e

INSTALL_DIR="/usr/local/bin"
SCRIPT_NAME="x"

echo "Installing x..."

# Check if running as root for /usr/local/bin
if [ "$INSTALL_DIR" = "/usr/local/bin" ] && [ "$(id -u)" -ne 0 ]; then
    echo "This script requires sudo to install to $INSTALL_DIR"
    echo "Please run: curl -sSL https://raw.githubusercontent.com/arpitbbhayani/x/master/install.sh | sudo sh"
    exit 1
fi

# Download the script
curl -sSL https://raw.githubusercontent.com/arpitbbhayani/x/master/x -o "$INSTALL_DIR/$SCRIPT_NAME"

# Make it executable
chmod +x "$INSTALL_DIR/$SCRIPT_NAME"

echo "✓ x installed successfully to $INSTALL_DIR/$SCRIPT_NAME"
echo ""
echo "Set your API key:"
echo "  export OPENAI_API_KEY=\"your-key\"     # or"
echo "  export ANTHROPIC_API_KEY=\"your-key\"  # or"
echo "  export GEMINI_API_KEY=\"your-key\"     # or"
echo "  export OLLAMA_MODEL=\"llama3.2\""
echo ""
echo "Usage: x <instruction>"
