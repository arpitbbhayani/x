# x

A natural language shell command executor.

- No external dependencies (just `curl` or `wget`)
- Supports OpenAI, Gemini, Anthropic, and OLLAMA (local) as providers
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
# Or use local OLLAMA
export OLLAMA_MODEL="llama3.2"  # any ollama model
```

Add to your shell config (`~/.bashrc`, `~/.zshrc`, etc):

```bash
echo 'export OPENAI_API_KEY="your-key"' >> ~/.bashrc
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

## Using OLLAMA (Local)

To use OLLAMA locally:

1. Install OLLAMA: https://ollama.ai
2. Start OLLAMA: `ollama serve`
3. Pull a model: `ollama pull llama3.2`
4. Set the model: `export OLLAMA_MODEL="llama3.2"`
5. Use `x` as normal

Optional: Set custom endpoint: `export OLLAMA_ENDPOINT="http://localhost:11434"`

## License

MIT
