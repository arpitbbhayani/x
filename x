#!/usr/bin/env bash

set -euo pipefail

# Resolve script directory (used to locate repo files like `VERSION` regardless of cwd).
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERSION_FILE="$SCRIPT_DIR/VERSION"

# --- CLI flags -------------------------------------------------------------
# Keep flag handling at the top so `x --help/--version/--upgrade` works without
# requiring any environment variables.
# --------------------------------------------------------------------------

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
	echo "  It supports OpenAI, Anthropic, Gemini, LM Studio, and Ollama API providers."
	echo "  Set one of: OPENAI_API_KEY, ANTHROPIC_API_KEY, GEMINI_API_KEY, LMSTUDIO_MODEL, or OLLAMA_MODEL"
	echo ""
	echo "Environment variables:"
	echo "  OpenAI:      OPENAI_API_KEY, OPENAI_MODEL"
	echo "  Anthropic:   ANTHROPIC_API_KEY, ANTHROPIC_MODEL"
	echo "  Gemini:      GEMINI_API_KEY, GEMINI_MODEL"
	echo "  LM Studio:   LMSTUDIO_MODEL (required), LMSTUDIO_BASE_URL (optional, default http://localhost:1234/), LMSTUDIO_API_KEY (optional)"
	echo "  Ollama:      OLLAMA_MODEL, OLLAMA_HOST (optional, default http://localhost:11434)"
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

	# Download latest version into a temporary directory.
	TEMP_DIR=$(mktemp -d)
	trap 'rm -rf "$TEMP_DIR"' EXIT

	echo "Downloading latest version..."
	if command -v curl &>/dev/null; then
		curl -L -o "$TEMP_DIR/install.sh" https://raw.githubusercontent.com/yourusername/x/main/install.sh
	elif command -v wget &>/dev/null; then
		wget -O "$TEMP_DIR/install.sh" https://raw.githubusercontent.com/yourusername/x/main/install.sh
	else
		echo "Error: Neither curl nor wget is available"
		exit 1
	fi

	# Run installation script.
	bash "$TEMP_DIR/install.sh"

	echo "Upgrade completed!"
	exit 0
fi

# --- Runtime options -------------------------------------------------------
# `--verbose` enables debug prints to stderr.
# --------------------------------------------------------------------------

# Enable debug mode if --verbose flag is passed
DEBUG=0
if [[ "${1:-}" == "--verbose" ]]; then
	DEBUG=1
	shift
fi

# --- Persistent config -----------------------------------------------------
# `~/.x/config` is a small shell snippet used to remember the last working
# model per provider (written as KEY="value").
# --------------------------------------------------------------------------

# Config directory
CONFIG_DIR="$HOME/.x"
CONFIG_FILE="$CONFIG_DIR/config"

# Create config directory if it doesn't exist
mkdir -p "$CONFIG_DIR"

# Load saved config (if present). This may set *_MODEL variables.
if [ -f "$CONFIG_FILE" ]; then
	# shellcheck disable=SC1090
	source "$CONFIG_FILE"
fi

# --- Provider selection ----------------------------------------------------
# Pick the first provider that is configured via env vars.
# Notes:
# - OpenAI/Anthropic/Gemini require API keys.
# - LM Studio uses an OpenAI-compatible local HTTP server; a model name is
#   required but an API key is optional.
# - Ollama uses its own local HTTP API; a model name is required.
# --------------------------------------------------------------------------

# Detect which API key is available
API_PROVIDER=""
if [ -n "${OPENAI_API_KEY:-}" ]; then
	API_PROVIDER="openai"
elif [ -n "${ANTHROPIC_API_KEY:-}" ]; then
	API_PROVIDER="anthropic"
elif [ -n "${GEMINI_API_KEY:-}" ]; then
	API_PROVIDER="gemini"
elif [ -n "${LMSTUDIO_MODEL:-}" ]; then
	API_PROVIDER="lmstudio"
elif [ -n "${OLLAMA_MODEL:-}" ]; then
	API_PROVIDER="ollama"
else
	echo "Error: No API key found. Set one of: OPENAI_API_KEY, ANTHROPIC_API_KEY, GEMINI_API_KEY, LMSTUDIO_MODEL, OLLAMA_MODEL"
	exit 1
fi

# Set default models if not configured.
# Users can override these via environment variables or persisted config.
if [ "$API_PROVIDER" = "openai" ] && [ -z "${OPENAI_MODEL:-}" ]; then
	OPENAI_MODEL="gpt-4o-mini"
fi
if [ "$API_PROVIDER" = "anthropic" ] && [ -z "${ANTHROPIC_MODEL:-}" ]; then
	ANTHROPIC_MODEL="claude-3-5-haiku-20241022"
fi
if [ "$API_PROVIDER" = "gemini" ] && [ -z "${GEMINI_MODEL:-}" ]; then
	GEMINI_MODEL="gemini-2.0-flash-exp"
fi

# --- Input ----------------------------------------------------------------
# Everything remaining on the command line is treated as the natural-language
# instruction sent to the model.
# --------------------------------------------------------------------------

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

# Combine all arguments into a single instruction string.
INSTRUCTION="$*"

# Detect available HTTP client (we support curl or wget).
if command -v curl &>/dev/null; then
	HTTP_CLIENT="curl"
elif command -v wget &>/dev/null; then
	HTTP_CLIENT="wget"
else
	echo "Error: Neither curl nor wget is available"
	exit 1
fi

# Build the prompt text.
# The model is instructed to return ONLY an executable shell command.
# NOTE: This is later placed into JSON; it must remain a single string.
PROMPT_TEXT="You are a shell command generator. Convert the user's natural language instruction into a shell command.\n\nRules:\n- Return ONLY the shell command, nothing else\n- No explanations, no markdown formatting, no code block markers\n- No backticks, no \`\`\`bash\`\`\`, no comments\n- Just the raw executable command(s)\n- Use pipes (|) and operators (&&, ||) as needed\n- If multiple commands are needed, combine them with && or ;\n\nContext:\n- Current directory: $(pwd)\n- Shell: ${SHELL}\n- OS: $(uname -s)\n\nInstruction: ${INSTRUCTION}\n\nCommand:"

[[ $DEBUG -eq 1 ]] && echo "DEBUG: Using API provider: $API_PROVIDER" >&2
[[ $DEBUG -eq 1 ]] && echo "DEBUG: Instruction: $INSTRUCTION" >&2

# --- Provider implementations ---------------------------------------------
# Each block:
# 1) builds a provider-specific JSON payload
# 2) sends the request via curl/wget
# 3) extracts the model output into $COMMAND
# 4) optionally persists the working model to `~/.x/config`
# --------------------------------------------------------------------------

# Make API request based on provider
if [ "$API_PROVIDER" = "openai" ]; then
	# Try models in order of preference (cheap to cheaper)
	OPENAI_MODELS=("${OPENAI_MODEL}" "gpt-4o-mini" "gpt-3.5-turbo")

	for MODEL in "${OPENAI_MODELS[@]}"; do
		[[ $DEBUG -eq 1 ]] && echo "DEBUG: Trying OpenAI model: $MODEL" >&2
		JSON_PAYLOAD=$(
			cat <<EOF
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

		# If the response contains an error, decide whether to fall back to the next
		# model or exit immediately.
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

		# Extract the generated command from JSON.
		# Prefer python for robust JSON parsing; fallback to a simple sed extraction.
		if command -v python3 &>/dev/null; then
			COMMAND=$(echo "$RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data['choices'][0]['message']['content'])" 2>/dev/null)
		else
			COMMAND=$(echo "$RESPONSE" | sed -n 's/.*"content"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
		fi

		# If we got a non-empty command, persist the working model and stop.
		if [ -n "$COMMAND" ]; then
			# Save working model to config
			echo "OPENAI_MODEL=\"$MODEL\"" >"$CONFIG_FILE"
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
		JSON_PAYLOAD=$(
			cat <<EOF
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

		# Model availability/validation errors may be recoverable by trying
		# the next model.
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

		# Extract the generated command.
			COMMAND=$(echo "$RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data['content'][0]['text'])" 2>/dev/null)
		else
			COMMAND=$(echo "$RESPONSE" | sed -n 's/.*"text"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
		fi

		# Persist the working model and stop.
		if [ -n "$COMMAND" ]; then
			# Save working model to config
			echo "ANTHROPIC_MODEL=\"$MODEL\"" >"$CONFIG_FILE"
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
		JSON_PAYLOAD=$(
			cat <<EOF
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

		# Gemini returns errors inside an "error" object.
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

		# Extract the generated command.
			COMMAND=$(echo "$RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data['candidates'][0]['content']['parts'][0]['text'])" 2>/dev/null)
		else
			COMMAND=$(echo "$RESPONSE" | sed -n 's/.*"text"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
		fi

		# Persist the working model and stop.
		if [ -n "$COMMAND" ]; then
			# Save working model to config
			echo "GEMINI_MODEL=\"$MODEL\"" >"$CONFIG_FILE"
			[[ $DEBUG -eq 1 ]] && echo "DEBUG: Saved working model: $MODEL" >&2
			[[ $DEBUG -eq 1 ]] && echo "DEBUG: Extracted command: $COMMAND" >&2
			break
		fi
	done

elif [ "$API_PROVIDER" = "lmstudio" ]; then
	MODEL="${LMSTUDIO_MODEL}"
	LMSTUDIO_BASE_URL="${LMSTUDIO_BASE_URL:-http://localhost:1234/v1}"

	[[ $DEBUG -eq 1 ]] && echo "DEBUG: Using LM Studio model: $MODEL at $LMSTUDIO_BASE_URL" >&2

	JSON_PAYLOAD=$(
		cat <<EOF
{
  "model": "$MODEL",
  "messages": [{"role": "user", "content": "PROMPT_PLACEHOLDER"}],
  "temperature": 0.1,
  "max_tokens": 500
}
EOF
	)
	JSON_PAYLOAD="${JSON_PAYLOAD//PROMPT_PLACEHOLDER/$PROMPT_TEXT}"

	[[ $DEBUG -eq 1 ]] && echo "DEBUG: Sending request to LM Studio..." >&2

	if [ "$HTTP_CLIENT" = "curl" ]; then
		if [ -n "${LMSTUDIO_API_KEY:-}" ]; then
			RESPONSE=$(curl -s -X POST "${LMSTUDIO_BASE_URL}/chat/completions" \
				-H "Content-Type: application/json" \
				-H "Authorization: Bearer ${LMSTUDIO_API_KEY}" \
				-d "$JSON_PAYLOAD")
		else
			RESPONSE=$(curl -s -X POST "${LMSTUDIO_BASE_URL}/chat/completions" \
				-H "Content-Type: application/json" \
				-d "$JSON_PAYLOAD")
		fi
	else
		if [ -n "${LMSTUDIO_API_KEY:-}" ]; then
			RESPONSE=$(wget -q -O- \
				--method=POST \
				--header="Content-Type: application/json" \
				--header="Authorization: Bearer ${LMSTUDIO_API_KEY}" \
				--body-data="$JSON_PAYLOAD" \
				"${LMSTUDIO_BASE_URL}/chat/completions")
		else
			RESPONSE=$(wget -q -O- \
				--method=POST \
				--header="Content-Type: application/json" \
				--body-data="$JSON_PAYLOAD" \
				"${LMSTUDIO_BASE_URL}/chat/completions")
		fi
	fi

	[[ $DEBUG -eq 1 ]] && echo "DEBUG: Response received" >&2
	[[ $DEBUG -eq 1 ]] && echo "DEBUG: Full response: $RESPONSE" >&2

	# Check for error in response.
	if echo "$RESPONSE" | grep -q '"error"'; then
		echo "Error: API request failed"
		echo "$RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); err = data.get('error'); print(err.get('message') if isinstance(err, dict) else (err or data))" 2>/dev/null || echo "$RESPONSE"
		exit 1
	fi

	# Extract the generated command.
	if command -v python3 &>/dev/null; then
		COMMAND=$(echo "$RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data['choices'][0]['message']['content'])" 2>/dev/null)
	else
		COMMAND=$(echo "$RESPONSE" | sed -n 's/.*"content"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
	fi

	# Persist last working LM Studio model.
	if [ -n "$COMMAND" ]; then
		echo "LMSTUDIO_MODEL=\"$MODEL\"" >"$CONFIG_FILE"
		[[ $DEBUG -eq 1 ]] && echo "DEBUG: Saved working model: $MODEL" >&2
		[[ $DEBUG -eq 1 ]] && echo "DEBUG: Extracted command: $COMMAND" >&2
	fi

elif [ "$API_PROVIDER" = "ollama" ]; then
	MODEL="${OLLAMA_MODEL}"
	OLLAMA_HOST="${OLLAMA_HOST:-http://localhost:11434}"

	[[ $DEBUG -eq 1 ]] && echo "DEBUG: Using Ollama model: $MODEL at $OLLAMA_HOST" >&2

	JSON_PAYLOAD=$(
		cat <<EOF
{
  "model": "$MODEL",
  "messages": [{"role": "user", "content": "PROMPT_PLACEHOLDER"}],
  "stream": false
}
EOF
	)
	JSON_PAYLOAD="${JSON_PAYLOAD//PROMPT_PLACEHOLDER/$PROMPT_TEXT}"

	[[ $DEBUG -eq 1 ]] && echo "DEBUG: Sending request to Ollama..." >&2

	if [ "$HTTP_CLIENT" = "curl" ]; then
		RESPONSE=$(curl -s -X POST "${OLLAMA_HOST}/api/chat" \
			-H "Content-Type: application/json" \
			-d "$JSON_PAYLOAD")
	else
		RESPONSE=$(wget -q -O- \
			--method=POST \
			--header="Content-Type: application/json" \
			--body-data="$JSON_PAYLOAD" \
			"${OLLAMA_HOST}/api/chat")
	fi

	[[ $DEBUG -eq 1 ]] && echo "DEBUG: Response received" >&2
	[[ $DEBUG -eq 1 ]] && echo "DEBUG: Full response: $RESPONSE" >&2

	# Check for error in response.
	if echo "$RESPONSE" | grep -q '"error"'; then
		echo "Error: API request failed"
		echo "$RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data.get('error', data))" 2>/dev/null || echo "$RESPONSE"
		exit 1
	fi

	# Extract the generated command.
	if command -v python3 &>/dev/null; then
		COMMAND=$(echo "$RESPONSE" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data['message']['content'])" 2>/dev/null)
	else
		COMMAND=$(echo "$RESPONSE" | sed -n 's/.*"content"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
	fi
fi

if [ -z "$COMMAND" ]; then
	# If we couldn't extract a command, print the raw response for debugging.
	echo "Error: Failed to generate command"
	echo "API Response: $RESPONSE"
	exit 1
fi

# --- Execution -------------------------------------------------------------
# Print the command, ask for confirmation, then execute.
# --------------------------------------------------------------------------

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
