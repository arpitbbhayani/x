package provider

import (
	"github.com/REDFOX1899/ask-sh/internal/config"
)

// Registry manages available providers and handles detection
type Registry struct {
	providers []Provider
	cfg       *config.Config
}

// NewRegistry creates a new provider registry with all providers
func NewRegistry(cfg *config.Config, verbose bool) *Registry {
	r := &Registry{
		cfg: cfg,
	}

	// Register providers in detection order
	r.providers = []Provider{
		NewOpenAI(cfg, verbose),
		NewAnthropic(cfg, verbose),
		NewGemini(cfg, verbose),
		NewOllama(cfg, verbose),
	}

	return r
}

// Detect returns the first available provider based on configuration
func (r *Registry) Detect() (Provider, error) {
	for _, p := range r.providers {
		if p.IsAvailable() {
			return p, nil
		}
	}
	return nil, ErrNoProvider
}

// Get returns a specific provider by name
func (r *Registry) Get(name string) (Provider, error) {
	for _, p := range r.providers {
		if p.Name() == name {
			if !p.IsAvailable() {
				return nil, ErrNoProvider
			}
			return p, nil
		}
	}
	return nil, ErrNoProvider
}

// List returns all registered providers
func (r *Registry) List() []Provider {
	return r.providers
}
