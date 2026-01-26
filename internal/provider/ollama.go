package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/REDFOX1899/ask-sh/internal/config"
)

// Ollama provider implementation (local LLM)
type Ollama struct {
	model   string
	host    string
	verbose bool
}

// Ollama API request/response types
type ollamaRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaResponse struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Error string `json:"error,omitempty"`
}

// NewOllama creates a new Ollama provider
func NewOllama(cfg *config.Config, verbose bool) *Ollama {
	return &Ollama{
		model:   cfg.OllamaModel,
		host:    cfg.OllamaHost,
		verbose: verbose,
	}
}

// Name returns the provider name
func (o *Ollama) Name() string {
	return "ollama"
}

// IsAvailable checks if Ollama is configured
func (o *Ollama) IsAvailable() bool {
	return o.model != ""
}

// GenerateCommand generates a shell command using Ollama
func (o *Ollama) GenerateCommand(ctx context.Context, prompt string) (*Response, error) {
	if o.verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Using Ollama model: %s at %s\n", o.model, o.host)
	}

	reqBody := ollamaRequest{
		Model: o.model,
		Messages: []ollamaMessage{
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	if o.verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Sending request to Ollama...\n")
	}

	url := fmt.Sprintf("%s/api/chat", o.host)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama at %s: %w", o.host, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if o.verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Response received\n")
		fmt.Fprintf(os.Stderr, "DEBUG: Full response: %s\n", string(body))
	}

	var result ollamaResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// Check for errors
	if result.Error != "" {
		return nil, fmt.Errorf("%w: %s", ErrAPIFailure, result.Error)
	}

	if result.Message.Content == "" {
		return nil, ErrEmptyResponse
	}

	command := strings.TrimSpace(result.Message.Content)
	if o.verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Extracted command: %s\n", command)
	}

	return &Response{
		Command:  command,
		Model:    o.model,
		Provider: o.Name(),
	}, nil
}

// ExplainCommand explains what a shell command does
func (o *Ollama) ExplainCommand(ctx context.Context, command string) (string, error) {
	prompt := fmt.Sprintf(`Explain this shell command in simple terms. Break down each flag and option.
Keep it concise but educational. Format as a brief explanation followed by a breakdown of flags.

Command: %s

Explanation:`, command)

	resp, err := o.callAPISimple(ctx, prompt)
	if err != nil {
		return "", err
	}
	return resp, nil
}

// RefineCommand refines a command based on user feedback
func (o *Ollama) RefineCommand(ctx context.Context, command, refinement string) (*Response, error) {
	prompt := fmt.Sprintf(`You are a shell command generator. Modify the given command based on the user's refinement request.

Current command: %s

User's refinement request: %s

Rules:
- Return ONLY the modified shell command, nothing else
- No explanations, no markdown formatting, no code block markers
- Just the raw executable command

Modified command:`, command, refinement)

	return o.GenerateCommand(ctx, prompt)
}

// callAPISimple makes a simple API call and returns just the text
func (o *Ollama) callAPISimple(ctx context.Context, prompt string) (string, error) {
	reqBody := ollamaRequest{
		Model: o.model,
		Messages: []ollamaMessage{
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/api/chat", o.host)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to Ollama at %s: %w", o.host, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result ollamaResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.Error != "" {
		return "", fmt.Errorf("%w: %s", ErrAPIFailure, result.Error)
	}

	if result.Message.Content == "" {
		return "", ErrEmptyResponse
	}

	return strings.TrimSpace(result.Message.Content), nil
}
