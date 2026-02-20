package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Use a nonexistent config path so we get defaults only
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "nonexistent.json")
	os.Setenv("GOGENTS_CONFIG", cfgPath)
	defer os.Unsetenv("GOGENTS_CONFIG")

	c, err := Load()
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	if c.OpenRouterURL != "https://openrouter.ai/api/v1" {
		t.Errorf("OpenRouterURL = %q, want openrouter default", c.OpenRouterURL)
	}
	if c.Model != "openrouter/free" {
		t.Errorf("Model = %q, want openrouter/free", c.Model)
	}
	if c.Workspace != "." {
		t.Errorf("Workspace = %q", c.Workspace)
	}
	if c.MaxIterations != 10 || c.MaxTokens != 8192 {
		t.Errorf("MaxIterations=%d MaxTokens=%d", c.MaxIterations, c.MaxTokens)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "nonexistent.json")
	os.Setenv("GOGENTS_CONFIG", cfgPath)
	os.Setenv("OPENROUTER_API_KEY", "sk-test")
	os.Setenv("GOGENTS_MODEL", "meta-llama/llama-3.3-70b-instruct")
	os.Setenv("GOGENTS_WORKSPACE", "/tmp/ws")
	os.Setenv("REDVECTOR_URL", "http://localhost:8888")
	os.Setenv("EMBED_API_URL", "http://localhost:11434/v1")
	os.Setenv("EMBED_MODEL", "arctic-embed")
	defer func() {
		os.Unsetenv("GOGENTS_CONFIG")
		os.Unsetenv("OPENROUTER_API_KEY")
		os.Unsetenv("GOGENTS_MODEL")
		os.Unsetenv("GOGENTS_WORKSPACE")
		os.Unsetenv("REDVECTOR_URL")
		os.Unsetenv("EMBED_API_URL")
		os.Unsetenv("EMBED_MODEL")
	}()

	c, err := Load()
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	if c.OpenRouterAPIKey != "sk-test" {
		t.Errorf("OpenRouterAPIKey = %q", c.OpenRouterAPIKey)
	}
	if c.Model != "meta-llama/llama-3.3-70b-instruct" {
		t.Errorf("Model = %q", c.Model)
	}
	if c.Workspace != "/tmp/ws" {
		t.Errorf("Workspace = %q", c.Workspace)
	}
	if c.RedVectorURL != "http://localhost:8888" {
		t.Errorf("RedVectorURL = %q", c.RedVectorURL)
	}
	if c.EmbedAPIURL != "http://localhost:11434/v1" {
		t.Errorf("EmbedAPIURL = %q", c.EmbedAPIURL)
	}
	if c.EmbedModel != "arctic-embed" {
		t.Errorf("EmbedModel = %q", c.EmbedModel)
	}
}

func TestLoad_OLLAMA_HOST(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("GOGENTS_CONFIG", filepath.Join(dir, "x.json"))
	os.Setenv("OLLAMA_HOST", "localhost:11434")
	defer func() {
		os.Unsetenv("GOGENTS_CONFIG")
		os.Unsetenv("OLLAMA_HOST")
	}()

	c, err := Load()
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	want := "http://localhost:11434/v1"
	if c.OpenRouterURL != want {
		t.Errorf("OpenRouterURL = %q, want %q", c.OpenRouterURL, want)
	}
}

func TestLoad_ConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	body := `{
		"openrouter_api_key": "sk-file",
		"model": "openrouter/auto",
		"workspace": "/home/agent",
		"max_iterations": 5,
		"max_tokens": 4096,
		"temperature": 0.5,
		"redvector_url": "http://127.0.0.1:8888",
		"embed_api_url": "http://127.0.0.1:11434/v1",
		"embed_model": "nomic-embed"
	}`
	if err := os.WriteFile(cfgPath, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	os.Setenv("GOGENTS_CONFIG", cfgPath)
	defer os.Unsetenv("GOGENTS_CONFIG")

	c, err := Load()
	if err != nil {
		t.Fatalf("Load() err = %v", err)
	}
	if c.OpenRouterAPIKey != "sk-file" {
		t.Errorf("OpenRouterAPIKey = %q", c.OpenRouterAPIKey)
	}
	if c.Model != "openrouter/auto" {
		t.Errorf("Model = %q", c.Model)
	}
	if c.Workspace != "/home/agent" {
		t.Errorf("Workspace = %q", c.Workspace)
	}
	if c.MaxIterations != 5 || c.MaxTokens != 4096 {
		t.Errorf("MaxIterations=%d MaxTokens=%d", c.MaxIterations, c.MaxTokens)
	}
	if c.Temperature != 0.5 {
		t.Errorf("Temperature = %v", c.Temperature)
	}
	if c.RedVectorURL != "http://127.0.0.1:8888" {
		t.Errorf("RedVectorURL = %q", c.RedVectorURL)
	}
	if c.EmbedModel != "nomic-embed" {
		t.Errorf("EmbedModel = %q", c.EmbedModel)
	}
}

func TestLoad_ConfigFileInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(cfgPath, []byte(`{invalid`), 0644); err != nil {
		t.Fatal(err)
	}
	os.Setenv("GOGENTS_CONFIG", cfgPath)
	defer os.Unsetenv("GOGENTS_CONFIG")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid JSON")
	}
}

func TestIsLocal(t *testing.T) {
	tests := []struct {
		url   string
		local bool
	}{
		{"http://localhost:11434/v1", true},
		{"http://127.0.0.1:11434/v1", true},
		{"https://localhost:11434/v1", true},
		{"https://openrouter.ai/api/v1", false},
		{"http://example.com", false},
	}
	for _, tt := range tests {
		c := &Config{OpenRouterURL: tt.url}
		got := c.IsLocal()
		if got != tt.local {
			t.Errorf("IsLocal(%q) = %v, want %v", tt.url, got, tt.local)
		}
	}
}
