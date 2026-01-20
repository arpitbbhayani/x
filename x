#!/usr/bin/env bash

set -euo pipefail

# Script directory (for accessing VERSION file)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERSION_FILE="$SCRIPT_DIR/VERSION"

# Handle --help flag
if [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "-h" ]]; then
    echo ""
    echo "Usage: x [--verbose] <instruction>"
    echo "       x --version"
    echo "       x --upgrade"
    echo "       x --help"
    echo ""
    echo "Example: x get all the git branches"
    echo ""
    echo "Options:"
    echo "  --verbose    Enable debug output"
    echo "  --version    Show version information"
    echo "  --upgrade    Upgrade to the latest version"
    echo "  --help, -h   Show this help message"
    echo ""
    echo "Description:"
    echo "  x converts natural language instructions into shell commands."
    echo "  It supports OpenAI, Anthropic, and Gemini API providers."
    echo "  Set one of: OPENAI_API_KEY, ANTHROPIC_API_KEY, or GEMINI_API_KEY or LOCAL_LLM"
    exit 0
fi

# Handle --version flag
if [[ "${1:-}" == "--version" ]]; then
    if [ -f "$VERSION_FILE" ]; then
        cat "$VERSION_FILE"
    else
        echo "Version file not found"
    fi
    exit 0
fi

# Handle --upgrade flag
if [[ "${1:-}" == "--upgrade" ]]; then
    echo "Upgrading x utility..."

    # Determine installation directory
    INSTALL_DIR="$SCRIPT_DIR"

    # Download latest version
    TEMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TEMP_DIR"' EXIT

    echo "Downloading latest version..."
    if command -v curl &> /dev/null; then
        curl -L -o "$TEMP_DIR/install.sh" https://raw.githubusercontent.com/yourusername/x/main/install.sh
    elif command -v wget &> /dev/null; then
        wget -O "$TEMP_DIR/install.sh" https://raw.githubusercontent.com/yourusername/x/main/install.sh
    else
        echo "Error: Neither curl nor wget is available"
        exit 1
    fi

    # Run installation script
    bash "$TEMP_DIR/install.sh"

    echo "Upgrade completed!"
    exit 0
fi

# Enable debug mode if --verbose flag is passed
DEBUG=0
if [[ "${1:-}" == "--verbose" ]]; then
    DEBUG=1
    shift
fi

# Config directory
CONFIG_DIR="$HOME/.x"
CONFIG_FILE="$CONFIG_DIR/config"

# Create config directory if it doesn't exist
mkdir -p "$CONFIG_DIR"

# Load saved config if exists
if [ -f "$CONFIG_FILE" ]; then
    source "$CONFIG_FILE"
fi

# Detect which API key is available
API_PROVIDER=""
if [ -n "${OPENAI_API_KEY:-}" ]; then
    API_PROVIDER="openai"
elif [ -n "${ANTHROPIC_API_KEY:-}" ]; then
    API_PROVIDER="anthropic"
elif [ -n "${GEMINI_API_KEY:-}" ]; then
    API_PROVIDER="gemini"
elif [ -n "${LOCAL_LLM:-}" ]; then
    API_PROVIDER="local"
else
    echo "Error: No API key found. Set one of: OPENAI_API_KEY, ANTHROPIC_API_KEY, GEMINI_API_KEY"
    exit 1
fi

# Set default models if not configured
if [ "$API_PROVIDER" = "openai" ] && [ -z "${OPENAI_MODEL:-}" ]; then
    OPENAI_MODEL="gpt-4o-mini"
fi
if [ "$API_PROVIDER" = "anthropic" ] && [ -z "${ANTHROPIC_MODEL:-}" ]; then
    ANTHROPIC_MODEL="claude-3-5-haiku-20241022"
fi
if [ "$API_PROVIDER" = "gemini" ] && [ -z "${GEMINI_MODEL:-}" ]; then
    GEMINI_MODEL="gemini-2.0-flash-exp"
fi
if [ "$API_PROVIDER" = "local" ] && [ -z "${LOCAL_MODEL:-}" ]; then
    LOCAL_MODEL=${LOCAL_LLM}
fi

# Check if instruction is provided
if [ $# -eq 0 ]; then
    echo "Usage: x [--verbose] <instruction>"
    echo "       x --version"
    echo "       x --upgrade"
    echo "       x --help"
    echo ""
    echo "Run 'x --help' for more information."
    exit 1
fi

# Combine all arguments into instruction
INSTRUCTION="$*"

# Detect available HTTP client
if command -v curl &> /dev/null; then
    HTTP_CLIENT="curl"
elif command -v wget &> /dev/null; then
    HTTP_CLIENT="wget"
else
    echo "Error: Neither curl nor wget is available"
    exit 1
fi

# Build system prompt (escape for JSON)
PROMPT_TEXT="You are a shell command generator. Convert the user's natural language instruction into a shell command.\n\nRules:\n- Return ONLY the shell command, nothing else\n- No explanations, no markdown formatting, no code block markers\n- No backticks, no \`\`\`bash\`\`\`, no comments\n- Just the raw executable command(s)\n- Use pipes (|) and operators (&&, ||) as needed\n- If multiple commands are needed, combine them with && or ;\n\nContext:\n- Current directory: $(pwd)\n- Shell: ${SHELL}\n- OS: $(uname -s)\n\nInstruction: ${INSTRUCTION}\n\nCommand:"

[[ $DEBUG -eq 1 ]] && echo "DEBUG: Using API provider: $API_PROVIDER" >&2
[[ $DEBUG -eq 1 ]] && echo "DEBUG: Instruction: $INSTRUCTION" >&2

# Make API request based on provider
if [ "$API_PROVIDER" = "openai" ]; then
    # Try models in order of preference (cheap to cheaper)
    OPENAI_MODELS=("${OPENAI_MODEL}" "gpt-4o-mini" "gpt-3.5-turbo")

    for MODEL in "${OPENAI_MODELS[@]}"; do
        [[ $DEBUG -eq 1 ]] && echo "DEBUG: Trying OpenAI model: $MODEL" >&2
        JSON_PAYLOAD=$(cat <<EOF
{
  "model": "$MODEL",
  "messages": [{"role": "user", "content": "PROMPT_PLACEHOLDER"}],
  "temperature": 0.1,
  "max_tokens": 500
}
EOF
)
        JSON_PAYLOAD="${JSON_PAYLOAD//PROMPT_PLACEHOLDER/$PROMPT_TEXT}"
        [[ $DEBUG -eq 1 ]] && echo "DEBUG: Sending request to OpenAI..." >&2
        if [ "$HTTP_CLIENT" = "curl" ]; then
            RESPONSE=$(curl -s -X POST https://api.openai.com/v1/chat/completions \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer ${OPENAI_API_KEY}" \
                -d "$JSON_PAYLOAD")
        else
            RESPONSE=$(wget -q -O- \
                --method=POST \
                --header="Content-Type: application/json" \
                --header="Authorization: Bearer ${OPENAI_API_KEY}" \
                --body-data="$JSON_PAYLOAD" \
                https://api.openai.com/v1/chat/completions)
        fi
        [[ $DEBUG -eq 1 ]] && echo "DEBUG: Response received" >&2
        [[ $DEBUG -eq 1 ]] && echo "DEBUG: Full response: $RESPONSE" >&2

        # Check for model-related errors
        if echo "$RESPONSE" | grep -q '"error"'; then
            ERROR_MSG=$(echo "$RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data.get('error', {}).get('code', ''))" 2>/dev/null)
            if [[ "$ERROR_MSG" == "model_not_found" ]] || echo "$RESPONSE" | grep -q "does not exist"; then
                [[ $DEBUG -eq 1 ]] && echo "DEBUG: Model $MODEL not available, trying next..." >&2
                continue
            else
                echo "Error: API request failed"
                echo "$RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data.get('error', {}).get('message', data))" 2>/dev/null || echo "$RESPONSE"
                exit 1
            fi
        fi

        if command -v python3 &> /dev/null; then
            COMMAND=$(echo "$RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data['choices'][0]['message']['content'])" 2>/dev/null)
        else
            COMMAND=$(echo "$RESPONSE" | sed -n 's/.*"content"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
        fi

        if [ -n "$COMMAND" ]; then
            # Save working model to config
            echo "OPENAI_MODEL=\"$MODEL\"" > "$CONFIG_FILE"
            [[ $DEBUG -eq 1 ]] && echo "DEBUG: Saved working model: $MODEL" >&2
            [[ $DEBUG -eq 1 ]] && echo "DEBUG: Extracted command: $COMMAND" >&2
            break
        fi
    done

elif [ "$API_PROVIDER" = "anthropic" ]; then
    # Try models in order of preference (cheap to cheaper)
    ANTHROPIC_MODELS=("${ANTHROPIC_MODEL}" "claude-3-5-haiku-20241022" "claude-3-haiku-20240307")

    for MODEL in "${ANTHROPIC_MODELS[@]}"; do
        [[ $DEBUG -eq 1 ]] && echo "DEBUG: Trying Anthropic model: $MODEL" >&2
        JSON_PAYLOAD=$(cat <<EOF
{
  "model": "$MODEL",
  "max_tokens": 500,
  "messages": [{"role": "user", "content": "PROMPT_PLACEHOLDER"}]
}
EOF
)
        JSON_PAYLOAD="${JSON_PAYLOAD//PROMPT_PLACEHOLDER/$PROMPT_TEXT}"
        [[ $DEBUG -eq 1 ]] && echo "DEBUG: Sending request to Anthropic..." >&2
        if [ "$HTTP_CLIENT" = "curl" ]; then
            RESPONSE=$(curl -s -X POST https://api.anthropic.com/v1/messages \
                -H "Content-Type: application/json" \
                -H "x-api-key: ${ANTHROPIC_API_KEY}" \
                -H "anthropic-version: 2023-06-01" \
                -d "$JSON_PAYLOAD")
        else
            RESPONSE=$(wget -q -O- \
                --method=POST \
                --header="Content-Type: application/json" \
                --header="x-api-key: ${ANTHROPIC_API_KEY}" \
                --header="anthropic-version: 2023-06-01" \
                --body-data="$JSON_PAYLOAD" \
                https://api.anthropic.com/v1/messages)
        fi
        [[ $DEBUG -eq 1 ]] && echo "DEBUG: Response received" >&2
        [[ $DEBUG -eq 1 ]] && echo "DEBUG: Full response: $RESPONSE" >&2

        # Check for model-related errors
        if echo "$RESPONSE" | grep -q '"error"'; then
            ERROR_TYPE=$(echo "$RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data.get('error', {}).get('type', ''))" 2>/dev/null)
            if [[ "$ERROR_TYPE" == "invalid_request_error" ]] && echo "$RESPONSE" | grep -q "model"; then
                [[ $DEBUG -eq 1 ]] && echo "DEBUG: Model $MODEL not available, trying next..." >&2
                continue
            else
                echo "Error: API request failed"
                echo "$RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data.get('error', {}).get('message', data))" 2>/dev/null || echo "$RESPONSE"
                exit 1
            fi
        fi

        if command -v python3 &> /dev/null; then
            COMMAND=$(echo "$RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data['content'][0]['text'])" 2>/dev/null)
        else
            COMMAND=$(echo "$RESPONSE" | sed -n 's/.*"text"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
        fi

        if [ -n "$COMMAND" ]; then
            # Save working model to config
            echo "ANTHROPIC_MODEL=\"$MODEL\"" > "$CONFIG_FILE"
            [[ $DEBUG -eq 1 ]] && echo "DEBUG: Saved working model: $MODEL" >&2
            [[ $DEBUG -eq 1 ]] && echo "DEBUG: Extracted command: $COMMAND" >&2
            break
        fi
    done

elif [ "$API_PROVIDER" = "gemini" ]; then
    # Try models in order of preference (cheap to cheaper)
    GEMINI_MODELS=("${GEMINI_MODEL}" "gemini-2.0-flash-exp" "gemini-1.5-flash" "gemini-pro")

    for MODEL in "${GEMINI_MODELS[@]}"; do
        [[ $DEBUG -eq 1 ]] && echo "DEBUG: Trying Gemini model: $MODEL" >&2
        JSON_PAYLOAD=$(cat <<EOF
{
  "contents": [{
    "parts": [{
      "text": "PROMPT_PLACEHOLDER"
    }]
  }],
  "generationConfig": {
    "temperature": 0.1,
    "maxOutputTokens": 500
  }
}
EOF
)
        JSON_PAYLOAD="${JSON_PAYLOAD//PROMPT_PLACEHOLDER/$PROMPT_TEXT}"
        [[ $DEBUG -eq 1 ]] && echo "DEBUG: Sending request to Gemini..." >&2
        if [ "$HTTP_CLIENT" = "curl" ]; then
            RESPONSE=$(curl -s -X POST \
                "https://generativelanguage.googleapis.com/v1beta/models/${MODEL}:generateContent?key=${GEMINI_API_KEY}" \
                -H 'Content-Type: application/json' \
                -d "$JSON_PAYLOAD")
        else
            RESPONSE=$(wget -q -O- \
                --method=POST \
                --header='Content-Type: application/json' \
                --body-data="$JSON_PAYLOAD" \
                "https://generativelanguage.googleapis.com/v1beta/models/${MODEL}:generateContent?key=${GEMINI_API_KEY}")
        fi
        [[ $DEBUG -eq 1 ]] && echo "DEBUG: Response received" >&2
        [[ $DEBUG -eq 1 ]] && echo "DEBUG: Full response: $RESPONSE" >&2

        # Check for model-related errors
        if echo "$RESPONSE" | grep -q '"error"'; then
            ERROR_CODE=$(echo "$RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data.get('error', {}).get('code', ''))" 2>/dev/null)
            if [[ "$ERROR_CODE" == "404" ]] || echo "$RESPONSE" | grep -q "not found"; then
                [[ $DEBUG -eq 1 ]] && echo "DEBUG: Model $MODEL not available, trying next..." >&2
                continue
            else
                echo "Error: API request failed"
                echo "$RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data.get('error', {}).get('message', data))" 2>/dev/null || echo "$RESPONSE"
                exit 1
            fi
        fi

        if command -v python3 &> /dev/null; then
            COMMAND=$(echo "$RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data['candidates'][0]['content']['parts'][0]['text'])" 2>/dev/null)
        else
            COMMAND=$(echo "$RESPONSE" | sed -n 's/.*"text"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
        fi

        if [ -n "$COMMAND" ]; then
            # Save working model to config
            echo "GEMINI_MODEL=\"$MODEL\"" > "$CONFIG_FILE"
            [[ $DEBUG -eq 1 ]] && echo "DEBUG: Saved working model: $MODEL" >&2
            [[ $DEBUG -eq 1 ]] && echo "DEBUG: Extracted command: $COMMAND" >&2
            break
        fi
    done
if [ "$API_PROVIDER" = "local" ]; then
    # Try models in order of preference (cheap to cheaper)
        [[ $DEBUG -eq 1 ]] && echo "DEBUG: Trying locally running model: $LOCAL_LLM" >&2
        JSON_PAYLOAD=$(cat <<EOF
{
  "model": "$LOCAL_LLM",
  "prompt": "PROMPT_PLACEHOLDER",
}
EOF
)
        JSON_PAYLOAD="${JSON_PAYLOAD//PROMPT_PLACEHOLDER/$PROMPT_TEXT}"
        [[ $DEBUG -eq 1 ]] && echo "DEBUG: Sending request to local llm..." >&2
        if [ "$HTTP_CLIENT" = "curl" ]; then
            RESPONSE=$(curl -s -X POST http://localhost:11434/api/generate \
                -H "Content-Type: application/json" \
                -d "$JSON_PAYLOAD")
        else
            RESPONSE=$(wget -q -O- \
                --method=POST \
                --header="Content-Type: application/json" \
                --body-data="$JSON_PAYLOAD" \
                http://localhost:11434/api/generate)
        fi
        [[ $DEBUG -eq 1 ]] && echo "DEBUG: Response received" >&2
        [[ $DEBUG -eq 1 ]] && echo "DEBUG: Full response: $RESPONSE" >&2

        # Check for model-related errors
        if echo "$RESPONSE" | grep -q '"error"'; then
            ERROR_MSG=$(echo "$RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data.get('error', {}).get('code', ''))" 2>/dev/null)
            if [[ "$ERROR_MSG" == "model_not_found" ]] || echo "$RESPONSE" | grep -q "does not exist"; then
                [[ $DEBUG -eq 1 ]] && echo "DEBUG: Model $MODEL not available, trying next..." >&2
                continue
            else
                echo "Error: API request failed"
                echo "$RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data.get('error', {}).get('message', data))" 2>/dev/null || echo "$RESPONSE"
                exit 1
            fi
        fi

        if command -v python3 &> /dev/null; then
            COMMAND=$(echo "$RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data['choices'][0]['message']['content'])" 2>/dev/null)
        else
            COMMAND=$(echo "$RESPONSE" | sed -n 's/.*"content"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
        fi

        if [ -n "$COMMAND" ]; then
            # Save working model to config
            echo "OPENAI_MODEL=\"$MODEL\"" > "$CONFIG_FILE"
            [[ $DEBUG -eq 1 ]] && echo "DEBUG: Saved working model: $MODEL" >&2
            [[ $DEBUG -eq 1 ]] && echo "DEBUG: Extracted command: $COMMAND" >&2
            break
        fi
    done
fi


if [ -z "$COMMAND" ]; then
    echo "Error: Failed to generate command"
    echo "API Response: $RESPONSE"
    exit 1
fi

# Display command and ask for confirmation
echo "----------"
echo -e "\033[1;33m>>>\033[0m $COMMAND"
read -p "Execute this command? (Y/n): " -n 1 -r
echo

if [[ $REPLY =~ ^[Nn]$ ]]; then
    echo "Command execution cancelled"
    exit 0
else
    eval "$COMMAND"
fi
