package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds all application configuration.
type Config struct {
	Transcription TranscriptionConfig `yaml:"transcription"`
	LLM           LLMConfig           `yaml:"llm"`
	Audio         AudioConfig         `yaml:"audio"`
	Paths         PathsConfig         `yaml:"paths"`
}

// TranscriptionConfig specifies which backend to use and its settings.
type TranscriptionConfig struct {
	Backend string             `yaml:"backend"`
	Local   LocalWhisperConfig `yaml:"local"`
	OpenAI  OpenAIConfig       `yaml:"openai"`
}

// LocalWhisperConfig contains settings for local whisper.cpp transcription.
type LocalWhisperConfig struct {
	ModelPath  string `yaml:"model_path"`
	BinaryPath string `yaml:"binary_path"`
	Threads    int    `yaml:"threads"`
	Language   string `yaml:"language"`
}

// OpenAIConfig contains settings for OpenAI Whisper API transcription.
type OpenAIConfig struct {
	Model    string `yaml:"model"`
	Language string `yaml:"language"`
}

// LLMConfig contains settings for LLM providers.
type LLMConfig struct {
	Backend string        `yaml:"backend"`
	Claude  ClaudeConfig  `yaml:"claude"`
	Ollama  OllamaConfig  `yaml:"ollama"`
}

// ClaudeConfig contains settings for Claude API.
type ClaudeConfig struct {
	Model     string `yaml:"model"`
	MaxTokens int    `yaml:"max_tokens"`
}

// OllamaConfig contains settings for Ollama API.
type OllamaConfig struct {
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
}

// AudioConfig contains audio recording settings.
type AudioConfig struct {
	SampleRate int `yaml:"sample_rate"`
	Channels   int `yaml:"channels"`
}

// PathsConfig contains file system paths.
type PathsConfig struct {
	SessionsDir string `yaml:"sessions_dir"`
	TempDir     string `yaml:"temp_dir"`
	PromptsDir  string `yaml:"prompts_dir"`
}

// Load reads configuration from ~/.config/trani/config.yaml.
// If the file doesn't exist, returns a Config with empty values.
// Callers should use ApplyDefaults() after Load() to set defaults.
func Load() (*Config, error) {
	configPath := filepath.Join(os.Getenv("HOME"), ".config", "trani", "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config YAML: %w", err)
	}

	return &cfg, nil
}

// ExpandPaths replaces ~ with $HOME in all path fields.
func (c *Config) ExpandPaths() {
	home := os.Getenv("HOME")

	c.Transcription.Local.ModelPath = expandPath(c.Transcription.Local.ModelPath, home)
	c.Transcription.Local.BinaryPath = expandPath(c.Transcription.Local.BinaryPath, home)
	c.Paths.SessionsDir = expandPath(c.Paths.SessionsDir, home)
	c.Paths.TempDir = expandPath(c.Paths.TempDir, home)
	c.Paths.PromptsDir = expandPath(c.Paths.PromptsDir, home)
}

func expandPath(path, home string) string {
	if path == "" {
		return path
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

// ApplyDefaults sets default values for empty configuration fields.
func (c *Config) ApplyDefaults() {
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "trani")

	if c.Paths.SessionsDir == "" {
		c.Paths.SessionsDir = filepath.Join(configDir, "sessions")
	}
	if c.Paths.TempDir == "" {
		c.Paths.TempDir = filepath.Join(configDir, "temp")
	}
	if c.Paths.PromptsDir == "" {
		c.Paths.PromptsDir = filepath.Join(configDir, "prompts")
	}

	if c.LLM.Backend == "" {
		c.LLM.Backend = "claude"
	}
	if c.LLM.Ollama.BaseURL == "" {
		c.LLM.Ollama.BaseURL = "http://localhost:11434"
	}
}
