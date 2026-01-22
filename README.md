# x

A natural language shell command executor.

- No external dependencies (just `curl` or `wget`)
- Supports OpenAI, Anthropic, Gemini, LM Studio, and Ollama as providers
- Shows command before execution for confirmation
- Automatically picks the best available model

## Installation

```bash
curl -sSL https://raw.githubusercontent.com/arpitbbhayani/x/master/install.sh | sudo sh
```

Set your API key (choose one):

```bash
export OPENAI_API_KEY="your-key"
export ANTHROPIC_API_KEY="your-key"
export GEMINI_API_KEY="your-key"
export LMSTUDIO_MODEL="your-model"                     # LM Studio (required)
export LMSTUDIO_BASE_URL="http://localhost:1234/v1"    # LM Studio (optional, default as shown)
export LMSTUDIO_API_KEY="your-key"                     # LM Studio (optional)
export OLLAMA_MODEL="llama3.2"
```

Add to your shell config (`~/.bashrc`, `~/.zshrc`, etc):

```bash
echo 'export OPENAI_API_KEY="your-key"' >> ~/.bashrc
```

## LM Studio (local) setup

- Install LM Studio: https://lmstudio.ai
- In LM Studio, download a chat/instruct model (e.g., "Meta Llama 3.1 8B Instruct", "Qwen2.5-Coder 7B", "Mistral 7B Instruct").
- Enable the OpenAI-compatible local server (Developer > Local Server). The default base URL is http://localhost:1234/v1.
- Exports to use it with x:

```bash
export LMSTUDIO_MODEL="your-model-name"                  # required, e.g. 'lmstudio-community/Meta-Llama-3.1-8B-Instruct'
export LMSTUDIO_BASE_URL="http://localhost:1234/v1"      # optional, default shown
export LMSTUDIO_API_KEY="your-key"                       # optional, only if you enabled API key auth
```

## Usage

```bash
x <instruction>
```

Examples:

```bash
x get all the git branches
x list all files modified in the last 7 days
x show disk usage of current directory
x count lines in all python files
```

The script generates a command and asks for confirmation before executing.

## License

MIT
