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

// OpenAI provider implementation
type OpenAI struct {
	apiKey  string
	models  []string
	verbose bool
}

// OpenAI API request/response types
type openAIRequest struct {
	Model       string           `json:"model"`
	Messages    []openAIMessage  `json:"messages"`
	Temperature float64          `json:"temperature"`
	MaxTokens   int              `json:"max_tokens"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *openAIError `json:"error,omitempty"`
}

type openAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// NewOpenAI creates a new OpenAI provider
func NewOpenAI(cfg *config.Config, verbose bool) *OpenAI {
	models := []string{cfg.OpenAIModel}
	// Add fallback models if not already the primary
	if cfg.OpenAIModel != "gpt-4o-mini" {
		models = append(models, "gpt-4o-mini")
	}
	if cfg.OpenAIModel != "gpt-3.5-turbo" {
		models = append(models, "gpt-3.5-turbo")
	}

	return &OpenAI{
		apiKey:  cfg.OpenAIAPIKey,
		models:  models,
		verbose: verbose,
	}
}

// Name returns the provider name
func (o *OpenAI) Name() string {
	return "openai"
}

// IsAvailable checks if OpenAI is configured
func (o *OpenAI) IsAvailable() bool {
	return o.apiKey != ""
}

// GenerateCommand generates a shell command using OpenAI
func (o *OpenAI) GenerateCommand(ctx context.Context, prompt string) (*Response, error) {
	var lastErr error

	for _, model := range o.models {
		if o.verbose {
			fmt.Fprintf(os.Stderr, "DEBUG: Trying OpenAI model: %s\n", model)
		}

		resp, err := o.callAPI(ctx, model, prompt)
		if err != nil {
			if err == ErrModelNotFound {
				if o.verbose {
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

func (o *OpenAI) callAPI(ctx context.Context, model, prompt string) (*Response, error) {
	reqBody := openAIRequest{
		Model: model,
		Messages: []openAIMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
		MaxTokens:   500,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	if o.verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Sending request to OpenAI...\n")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

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

	if o.verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Response received\n")
		fmt.Fprintf(os.Stderr, "DEBUG: Full response: %s\n", string(body))
	}

	var result openAIResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// Check for errors
	if result.Error != nil {
		if result.Error.Code == "model_not_found" || strings.Contains(result.Error.Message, "does not exist") {
			return nil, ErrModelNotFound
		}
		return nil, fmt.Errorf("%w: %s", ErrAPIFailure, result.Error.Message)
	}

	if len(result.Choices) == 0 || result.Choices[0].Message.Content == "" {
		return nil, ErrEmptyResponse
	}

	command := strings.TrimSpace(result.Choices[0].Message.Content)
	if o.verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Extracted command: %s\n", command)
	}

	return &Response{
		Command:  command,
		Model:    model,
		Provider: o.Name(),
	}, nil
}

// ExplainCommand explains what a shell command does
func (o *OpenAI) ExplainCommand(ctx context.Context, command string) (string, error) {
	prompt := fmt.Sprintf(`Explain this shell command in simple terms. Break down each flag and option.
Keep it concise but educational. Format as a brief explanation followed by a breakdown of flags.

Command: %s

Explanation:`, command)

	resp, err := o.callAPISimple(ctx, o.models[0], prompt, 800)
	if err != nil {
		return "", err
	}
	return resp, nil
}

// RefineCommand refines a command based on user feedback
func (o *OpenAI) RefineCommand(ctx context.Context, command, refinement string) (*Response, error) {
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
func (o *OpenAI) callAPISimple(ctx context.Context, model, prompt string, maxTokens int) (string, error) {
	reqBody := openAIRequest{
		Model: model,
		Messages: []openAIMessage{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   maxTokens,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

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

	var result openAIResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", fmt.Errorf("%w: %s", ErrAPIFailure, result.Error.Message)
	}

	if len(result.Choices) == 0 || result.Choices[0].Message.Content == "" {
		return "", ErrEmptyResponse
	}

	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}
