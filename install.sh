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
echo "Configure a provider :"
echo "  export OPENAI_API_KEY=\"your-key\"                     # OpenAI"
echo "  export ANTHROPIC_API_KEY=\"your-key\"                  # Anthropic"
echo "  export GEMINI_API_KEY=\"your-key\"                     # Gemini"
echo "  export LMSTUDIO_MODEL=\"your-model\"                   # LM Studio (required)"
echo "  export LMSTUDIO_BASE_URL=\"http://localhost:1234/v1\"  # LM Studio (optional, default as shown)"
echo "  export LMSTUDIO_API_KEY=\"your-key\"                   # LM Studio (optional)"
echo "  export OLLAMA_MODEL=\"llama3.2\"                       # Ollama"
echo ""
echo "Usage: x <instruction>"
