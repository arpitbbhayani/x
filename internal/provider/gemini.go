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

// Gemini provider implementation
type Gemini struct {
	apiKey  string
	models  []string
	verbose bool
}

// Gemini API request/response types
type geminiRequest struct {
	Contents         []geminiContent  `json:"contents"`
	GenerationConfig geminiGenConfig  `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenConfig struct {
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *geminiError `json:"error,omitempty"`
}

type geminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// NewGemini creates a new Gemini provider
func NewGemini(cfg *config.Config, verbose bool) *Gemini {
	models := []string{cfg.GeminiModel}
	// Add fallback models if not already the primary
	if cfg.GeminiModel != "gemini-2.0-flash-exp" {
		models = append(models, "gemini-2.0-flash-exp")
	}
	if cfg.GeminiModel != "gemini-1.5-flash" {
		models = append(models, "gemini-1.5-flash")
	}
	if cfg.GeminiModel != "gemini-pro" {
		models = append(models, "gemini-pro")
	}

	return &Gemini{
		apiKey:  cfg.GeminiAPIKey,
		models:  models,
		verbose: verbose,
	}
}

// Name returns the provider name
func (g *Gemini) Name() string {
	return "gemini"
}

// IsAvailable checks if Gemini is configured
func (g *Gemini) IsAvailable() bool {
	return g.apiKey != ""
}

// GenerateCommand generates a shell command using Gemini
func (g *Gemini) GenerateCommand(ctx context.Context, prompt string) (*Response, error) {
	var lastErr error

	for _, model := range g.models {
		if g.verbose {
			fmt.Fprintf(os.Stderr, "DEBUG: Trying Gemini model: %s\n", model)
		}

		resp, err := g.callAPI(ctx, model, prompt)
		if err != nil {
			if err == ErrModelNotFound {
				if g.verbose {
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

func (g *Gemini) callAPI(ctx context.Context, model, prompt string) (*Response, error) {
	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: geminiGenConfig{
			Temperature:     0.1,
			MaxOutputTokens: 500,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	if g.verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Sending request to Gemini...\n")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

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

	if g.verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Response received\n")
		fmt.Fprintf(os.Stderr, "DEBUG: Full response: %s\n", string(body))
	}

	var result geminiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// Check for errors
	if result.Error != nil {
		if result.Error.Code == 404 || strings.Contains(result.Error.Message, "not found") {
			return nil, ErrModelNotFound
		}
		return nil, fmt.Errorf("%w: %s", ErrAPIFailure, result.Error.Message)
	}

	if len(result.Candidates) == 0 ||
		len(result.Candidates[0].Content.Parts) == 0 ||
		result.Candidates[0].Content.Parts[0].Text == "" {
		return nil, ErrEmptyResponse
	}

	command := strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text)
	if g.verbose {
		fmt.Fprintf(os.Stderr, "DEBUG: Extracted command: %s\n", command)
	}

	return &Response{
		Command:  command,
		Model:    model,
		Provider: g.Name(),
	}, nil
}

// ExplainCommand explains what a shell command does
func (g *Gemini) ExplainCommand(ctx context.Context, command string) (string, error) {
	prompt := fmt.Sprintf(`Explain this shell command in simple terms. Break down each flag and option.
Keep it concise but educational. Format as a brief explanation followed by a breakdown of flags.

Command: %s

Explanation:`, command)

	resp, err := g.callAPISimple(ctx, g.models[0], prompt, 800)
	if err != nil {
		return "", err
	}
	return resp, nil
}

// RefineCommand refines a command based on user feedback
func (g *Gemini) RefineCommand(ctx context.Context, command, refinement string) (*Response, error) {
	prompt := fmt.Sprintf(`You are a shell command generator. Modify the given command based on the user's refinement request.

Current command: %s

User's refinement request: %s

Rules:
- Return ONLY the modified shell command, nothing else
- No explanations, no markdown formatting, no code block markers
- Just the raw executable command

Modified command:`, command, refinement)

	return g.GenerateCommand(ctx, prompt)
}

// callAPISimple makes a simple API call and returns just the text
func (g *Gemini) callAPISimple(ctx context.Context, model, prompt string, maxTokens int) (string, error) {
	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: geminiGenConfig{
			Temperature:     0.3,
			MaxOutputTokens: maxTokens,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

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

	var result geminiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.Error != nil {
		return "", fmt.Errorf("%w: %s", ErrAPIFailure, result.Error.Message)
	}

	if len(result.Candidates) == 0 ||
		len(result.Candidates[0].Content.Parts) == 0 ||
		result.Candidates[0].Content.Parts[0].Text == "" {
		return "", ErrEmptyResponse
	}

	return strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text), nil
}
