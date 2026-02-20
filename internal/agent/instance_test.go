package agent

import (
	"path/filepath"
	"testing"
)

func TestNewInstance_Defaults(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Workspace:     dir,
		OpenRouterKey: "key",
		OpenRouterURL: "https://openrouter.ai/api/v1",
	}
	a := NewInstance(cfg)
	if a.ID != "main" {
		t.Errorf("ID = %q", a.ID)
	}
	if a.Model != "openrouter/auto" {
		t.Errorf("Model = %q (empty config should default)", a.Model)
	}
	if a.Workspace == "" {
		t.Error("Workspace empty")
	}
	abs, _ := filepath.Abs(dir)
	if a.Workspace != abs {
		t.Errorf("Workspace = %q, want %q", a.Workspace, abs)
	}
	if a.MaxIterations != 10 {
		t.Errorf("MaxIterations = %d", a.MaxIterations)
	}
	if a.MaxTokens != 8192 {
		t.Errorf("MaxTokens = %d", a.MaxTokens)
	}
	if a.Temperature != 0.7 {
		t.Errorf("Temperature = %v", a.Temperature)
	}
	if a.Client == nil {
		t.Error("Client nil")
	}
	if a.Tools == nil {
		t.Error("Tools nil")
	}
	if a.Instructions == "" {
		t.Error("Instructions empty")
	}
}

func TestNewInstance_ModelFromConfig(t *testing.T) {
	cfg := Config{
		Workspace:     t.TempDir(),
		Model:         "openrouter/free",
		OpenRouterKey: "key",
		OpenRouterURL: "https://openrouter.ai/api/v1",
	}
	a := NewInstance(cfg)
	if a.Model != "openrouter/free" {
		t.Errorf("Model = %q", a.Model)
	}
}

func TestNewInstance_ToolsRegistered(t *testing.T) {
	cfg := Config{
		Workspace:     t.TempDir(),
		OpenRouterKey: "key",
		OpenRouterURL: "https://openrouter.ai/api/v1",
	}
	a := NewInstance(cfg)
	list := a.Tools.List()
	expected := map[string]bool{
		"read_file":  true,
		"write_file": true,
		"list_dir":   true,
		"run_shell":  true,
		"web_fetch":  true,
	}
	for _, name := range list {
		if !expected[name] && name != "rag_search" {
			t.Errorf("unexpected tool %q", name)
		}
	}
	if _, ok := a.Tools.Get("read_file"); !ok {
		t.Error("read_file not registered")
	}
	if _, ok := a.Tools.Get("web_fetch"); !ok {
		t.Error("web_fetch not registered")
	}
}

func TestNewInstance_WithRedVector_RegistersRAG(t *testing.T) {
	cfg := Config{
		Workspace:     t.TempDir(),
		OpenRouterKey: "key",
		OpenRouterURL: "https://openrouter.ai/api/v1",
		RedVectorURL:  "http://localhost:8888",
		EmbedAPIURL:   "http://localhost:11434/v1",
		EmbedModel:    "arctic-embed",
	}
	a := NewInstance(cfg)
	if _, ok := a.Tools.Get("rag_search"); !ok {
		t.Error("rag_search not registered when RedVectorURL and EmbedAPIURL set")
	}
}

func TestNewInstance_ToProviderDefs(t *testing.T) {
	cfg := Config{
		Workspace:     t.TempDir(),
		OpenRouterKey: "key",
		OpenRouterURL: "https://openrouter.ai/api/v1",
	}
	a := NewInstance(cfg)
	defs := a.Tools.ToProviderDefs()
	if len(defs) < 5 {
		t.Errorf("ToProviderDefs() len = %d", len(defs))
	}
	names := make(map[string]bool)
	for _, d := range defs {
		names[d.Function.Name] = true
	}
	if !names["read_file"] || !names["run_shell"] {
		t.Errorf("defs = %v", names)
	}
	// Cover Instance.ToProviderDefs() wrapper
	defs2 := a.ToProviderDefs()
	if len(defs2) != len(defs) {
		t.Errorf("ToProviderDefs() wrapper len = %d, want %d", len(defs2), len(defs))
	}
}
