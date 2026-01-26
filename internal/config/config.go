package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	// Provider settings
	Provider string `mapstructure:"provider"`

	// OpenAI settings
	OpenAIAPIKey string `mapstructure:"openai_api_key"`
	OpenAIModel  string `mapstructure:"openai_model"`

	// Anthropic settings
	AnthropicAPIKey string `mapstructure:"anthropic_api_key"`
	AnthropicModel  string `mapstructure:"anthropic_model"`

	// Gemini settings
	GeminiAPIKey string `mapstructure:"gemini_api_key"`
	GeminiModel  string `mapstructure:"gemini_model"`

	// Ollama settings
	OllamaModel string `mapstructure:"ollama_model"`
	OllamaHost  string `mapstructure:"ollama_host"`

	// General settings
	Verbose bool `mapstructure:"verbose"`
	Timeout int  `mapstructure:"timeout"`
}

// Default model values
const (
	DefaultOpenAIModel    = "gpt-4o-mini"
	DefaultAnthropicModel = "claude-3-5-haiku-20241022"
	DefaultGeminiModel    = "gemini-2.0-flash-exp"
	DefaultOllamaHost     = "http://localhost:11434"
	DefaultTimeout        = 30
)

// Manager handles configuration loading and saving
type Manager struct {
	v       *viper.Viper
	cfgDir  string
	cfgPath string
}

// NewManager creates a new configuration manager
func NewManager() (*Manager, error) {
	v := viper.New()

	// Get config directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	cfgDir := filepath.Join(homeDir, ".x")
	cfgPath := filepath.Join(cfgDir, "config.yaml")

	// Config file settings
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(cfgDir)
	v.AddConfigPath(".")

	// Environment variable binding - no prefix, use exact names
	v.AutomaticEnv()

	// Map environment variables to config keys
	v.BindEnv("openai_api_key", "OPENAI_API_KEY")
	v.BindEnv("anthropic_api_key", "ANTHROPIC_API_KEY")
	v.BindEnv("gemini_api_key", "GEMINI_API_KEY")
	v.BindEnv("ollama_model", "OLLAMA_MODEL")
	v.BindEnv("ollama_host", "OLLAMA_HOST")

	// Set defaults
	v.SetDefault("ollama_host", DefaultOllamaHost)
	v.SetDefault("timeout", DefaultTimeout)
	v.SetDefault("openai_model", DefaultOpenAIModel)
	v.SetDefault("anthropic_model", DefaultAnthropicModel)
	v.SetDefault("gemini_model", DefaultGeminiModel)

	// Read config file (ignore if not found)
	_ = v.ReadInConfig()

	return &Manager{v: v, cfgDir: cfgDir, cfgPath: cfgPath}, nil
}

// Load returns the current configuration
func (m *Manager) Load() (*Config, error) {
	cfg := &Config{}
	if err := m.v.Unmarshal(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// SaveWorkingModel saves the working model for a provider
func (m *Manager) SaveWorkingModel(provider, model string) error {
	key := provider + "_model"
	m.v.Set(key, model)

	// Ensure directory exists
	if err := os.MkdirAll(m.cfgDir, 0755); err != nil {
		return err
	}

	return m.v.WriteConfigAs(m.cfgPath)
}

// GetViper returns the underlying viper instance for flag binding
func (m *Manager) GetViper() *viper.Viper {
	return m.v
}
