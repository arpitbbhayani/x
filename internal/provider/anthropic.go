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

// Anthropic provider implementation
type Anthropic struct {
	apiKey  string
	models  []string
	verbose bool
}

// Anthropic API request/response types
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *anthropicError `json:"error,omitempty"`
}

type anthropicError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// NewAnthropic creates a new Anthropic provider
func NewAnthropic(cfg *config.Config, verbose bool) *Anthropic {
	models := []string{cfg.AnthropicModel}
	// Add fallback models if not already the primary
	if cfg.AnthropicModel != "claude-3-5-haiku-20241022" {
		models = append(models, "claude-3-5-haiku-20241022")
	}
	if cfg.AnthropicModel != "claude-3-haiku-20240307" {
		models = append(models, "claude-3-haiku-20240307")
	}

	return &Anthropic{
		apiKey:  cfg.AnthropicAPIKey,
		models:  models,
		verbose: verbose,
	}
}

// Name returns the provider name
func (a *Anthropic) Name() string {
	return "anthropic"
}

// IsAvailable checks if Anthropic is configured
func (a *Anthropic) IsAvailable() bool {
	return a.apiKey != ""
}

// GenerateCommand generates a shell command using Anthropic
func (a *Anthropic) GenerateCommand(ctx context.Context, prompt string) (*Response, error) {
	var lastErr error

	for _, model := range a.models {
		if a.verbose {
			fmt.Fprintf(os.Stderr, "DEBUG: Trying Anthropic model: %s\n", model)
		}

		resp, err := a.callAPI(ctx, model, prompt)
		if err != nil {
			if err == ErrModelNotFound {
				if a.verbose {
					fmt.Fprintf(os.Stderr, "DEBUG: Model %s not available, trying next...\n", model)
				}
				lastErr = err
				continue
			}
			return nil, err
		}

		return resp, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, ErrAPIFailure
}

func (a *Anthropic) callAPI(ctx context.Context, model, prompt string) (*Response, error) {
	reqBody := anthropicRequest{
		Model:     model,
		MaxTokens: 500,
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	if a.verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Sending request to Anthropic...\n")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if a.verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Response received\n")
		fmt.Fprintf(os.Stderr, "DEBUG: Full response: %s\n", string(body))
	}

	var result anthropicResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// Check for errors
	if result.Error != nil {
		if result.Error.Type == "invalid_request_error" && strings.Contains(result.Error.Message, "model") {
			return nil, ErrModelNotFound
		}
		return nil, fmt.Errorf("%w: %s", ErrAPIFailure, result.Error.Message)
	}

	if len(result.Content) == 0 || result.Content[0].Text == "" {
		return nil, ErrEmptyResponse
	}

	command := strings.TrimSpace(result.Content[0].Text)
	if a.verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Extracted command: %s\n", command)
	}

	return &Response{
		Command:  command,
		Model:    model,
		Provider: a.Name(),
	}, nil
}

// ExplainCommand explains what a shell command does
func (a *Anthropic) ExplainCommand(ctx context.Context, command string) (string, error) {
	prompt := fmt.Sprintf(`Explain this shell command in simple terms. Break down each flag and option.
Keep it concise but educational. Format as a brief explanation followed by a breakdown of flags.

Command: %s

Explanation:`, command)

	resp, err := a.callAPISimple(ctx, a.models[0], prompt, 800)
	if err != nil {
		return "", err
	}
	return resp, nil
}

// RefineCommand refines a command based on user feedback
func (a *Anthropic) RefineCommand(ctx context.Context, command, refinement string) (*Response, error) {
	prompt := fmt.Sprintf(`You are a shell command generator. Modify the given command based on the user's refinement request.

Current command: %s

User's refinement request: %s

Rules:
- Return ONLY the modified shell command, nothing else
- No explanations, no markdown formatting, no code block markers
- Just the raw executable command

Modified command:`, command, refinement)

	return a.GenerateCommand(ctx, prompt)
}

// callAPISimple makes a simple API call and returns just the text
func (a *Anthropic) callAPISimple(ctx context.Context, model, prompt string, maxTokens int) (string, error) {
	reqBody := anthropicRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result anthropicResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", fmt.Errorf("%w: %s", ErrAPIFailure, result.Error.Message)
	}

	if len(result.Content) == 0 || result.Content[0].Text == "" {
		return "", ErrEmptyResponse
	}

	return strings.TrimSpace(result.Content[0].Text), nil
}
