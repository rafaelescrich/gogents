package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds agent configuration (env + file).
type Config struct {
	// LLM: OpenRouter (cloud) or Ollama/local (OpenAI-compat at e.g. http://localhost:11434/v1)
	OpenRouterAPIKey string `json:"openrouter_api_key,omitempty"`
	OpenRouterURL   string `json:"openrouter_url,omitempty"` // also used for Ollama: http://localhost:11434/v1
	Model           string `json:"model,omitempty"`

	// Agent
	Workspace     string  `json:"workspace,omitempty"`
	MaxIterations int     `json:"max_iterations,omitempty"`
	MaxTokens     int     `json:"max_tokens,omitempty"`
	Temperature   float64 `json:"temperature,omitempty"`
	Instructions  string  `json:"instructions,omitempty"`

	// RAG (RedVector)
	RedVectorURL string `json:"redvector_url,omitempty"`
	EmbedAPIURL  string `json:"embed_api_url,omitempty"`
	EmbedAPIKey  string `json:"embed_api_key,omitempty"`
	EmbedModel   string `json:"embed_model,omitempty"`
}

// Load reads config from env and optional config file.
// Config file path: GOGENTS_CONFIG or ~/.gogents/config.json
func Load() (*Config, error) {
	c := &Config{
		OpenRouterURL: "https://openrouter.ai/api/v1",
		Model:         "openrouter/free", // free tier by default; use openrouter/auto for best paid
		Workspace:     ".",
		MaxIterations: 10,
		MaxTokens:     8192,
		Temperature:   0.7,
	}

	// Env overrides
	if v := os.Getenv("OPENROUTER_API_KEY"); v != "" {
		c.OpenRouterAPIKey = v
	}
	if v := os.Getenv("OPENROUTER_URL"); v != "" {
		c.OpenRouterURL = strings.TrimRight(v, "/")
	}
	if v := os.Getenv("OLLAMA_HOST"); v != "" {
		// OLLAMA_HOST is e.g. localhost:11434; we need /v1 for OpenAI-compat
		c.OpenRouterURL = "http://" + strings.TrimPrefix(strings.TrimPrefix(v, "http://"), "https://")
		if !strings.HasSuffix(c.OpenRouterURL, "/v1") {
			c.OpenRouterURL = strings.TrimRight(c.OpenRouterURL, "/") + "/v1"
		}
	}
	if v := os.Getenv("LLM_BASE_URL"); v != "" {
		c.OpenRouterURL = strings.TrimRight(v, "/")
	}
	if v := os.Getenv("GOGENTS_MODEL"); v != "" {
		c.Model = v
	}
	if v := os.Getenv("GOGENTS_WORKSPACE"); v != "" {
		c.Workspace = v
	}
	if v := os.Getenv("REDVECTOR_URL"); v != "" {
		c.RedVectorURL = strings.TrimRight(v, "/")
	}
	if v := os.Getenv("EMBED_API_URL"); v != "" {
		c.EmbedAPIURL = strings.TrimRight(v, "/")
	}
	if v := os.Getenv("EMBED_API_KEY"); v != "" {
		c.EmbedAPIKey = v
	}
	if v := os.Getenv("EMBED_MODEL"); v != "" {
		c.EmbedModel = v
	}

	configPath := os.Getenv("GOGENTS_CONFIG")
	if configPath == "" {
		home, _ := os.UserHomeDir()
		configPath = filepath.Join(home, ".gogents", "config.json")
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return nil, fmt.Errorf("read config %s: %w", configPath, err)
	}
	var file struct {
		OpenRouterAPIKey string  `json:"openrouter_api_key,omitempty"`
		OpenRouterURL    string  `json:"openrouter_url,omitempty"`
		OllamaHost       string  `json:"ollama_host,omitempty"` // e.g. localhost:11434 → sets openrouter_url to http://.../v1
		LLMBaseURL       string  `json:"llm_base_url,omitempty"`
		Model            string  `json:"model,omitempty"`
		Workspace        string  `json:"workspace,omitempty"`
		MaxIterations    int     `json:"max_iterations,omitempty"`
		MaxTokens        int     `json:"max_tokens,omitempty"`
		Temperature      float64 `json:"temperature,omitempty"`
		Instructions     string  `json:"instructions,omitempty"`
		RedVectorURL     string  `json:"redvector_url,omitempty"`
		EmbedAPIURL      string  `json:"embed_api_url,omitempty"`
		EmbedAPIKey      string  `json:"embed_api_key,omitempty"`
		EmbedModel       string  `json:"embed_model,omitempty"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if file.OpenRouterAPIKey != "" {
		c.OpenRouterAPIKey = file.OpenRouterAPIKey
	}
	if file.OpenRouterURL != "" {
		c.OpenRouterURL = file.OpenRouterURL
	}
	if file.OllamaHost != "" {
		h := strings.TrimPrefix(strings.TrimPrefix(file.OllamaHost, "http://"), "https://")
		c.OpenRouterURL = "http://" + strings.TrimRight(h, "/") + "/v1"
	}
	if file.LLMBaseURL != "" {
		c.OpenRouterURL = strings.TrimRight(file.LLMBaseURL, "/")
	}
	if file.Model != "" {
		c.Model = file.Model
	}
	if file.Workspace != "" {
		c.Workspace = file.Workspace
	}
	if file.MaxIterations > 0 {
		c.MaxIterations = file.MaxIterations
	}
	if file.MaxTokens > 0 {
		c.MaxTokens = file.MaxTokens
	}
	if file.Temperature != 0 {
		c.Temperature = file.Temperature
	}
	if file.Instructions != "" {
		c.Instructions = file.Instructions
	}
	if file.RedVectorURL != "" {
		c.RedVectorURL = file.RedVectorURL
	}
	if file.EmbedAPIURL != "" {
		c.EmbedAPIURL = file.EmbedAPIURL
	}
	if file.EmbedAPIKey != "" {
		c.EmbedAPIKey = file.EmbedAPIKey
	}
	if file.EmbedModel != "" {
		c.EmbedModel = file.EmbedModel
	}
	return c, nil
}

// IsLocal returns true if the configured LLM URL is local (Ollama/local backend).
// When true, API key is optional (Ollama ignores it).
func (c *Config) IsLocal() bool {
	u := strings.ToLower(c.OpenRouterURL)
	return strings.Contains(u, "localhost") || strings.Contains(u, "127.0.0.1") ||
		strings.HasPrefix(u, "http://localhost") || strings.HasPrefix(u, "http://127.0.0.1")
}

