package provider

import (
	"context"
	"errors"
)

// Common errors
var (
	ErrNoProvider    = errors.New("no API provider configured. Set one of: OPENAI_API_KEY, ANTHROPIC_API_KEY, GEMINI_API_KEY, or OLLAMA_MODEL")
	ErrModelNotFound = errors.New("model not found")
	ErrAPIFailure    = errors.New("API request failed")
	ErrEmptyResponse = errors.New("empty response from API")
)

// Provider defines the interface for AI providers
type Provider interface {
	// Name returns the provider identifier (e.g., "openai", "anthropic")
	Name() string

	// GenerateCommand sends a prompt and returns the generated shell command
	GenerateCommand(ctx context.Context, prompt string) (*Response, error)

	// ExplainCommand explains what a command does
	ExplainCommand(ctx context.Context, command string) (string, error)

	// RefineCommand refines a command based on user feedback
	RefineCommand(ctx context.Context, command, refinement string) (*Response, error)

	// IsAvailable checks if the provider can be used (has required config)
	IsAvailable() bool
}

// Response contains the generated command and metadata
type Response struct {
	Command  string // The generated shell command
	Model    string // Which model was used
	Provider string // Which provider was used
}
