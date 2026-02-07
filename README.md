# x

A natural language shell command executor.

- No external dependencies (just `curl` or `wget`)
- Supports OpenAI, Gemini, Anthropic, and Ollama as providers
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
export OLLAMA_MODEL="llama3.2"
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

## License

MIT
