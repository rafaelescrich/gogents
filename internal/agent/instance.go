package agent

import (
	"context"
	"os"
	"path/filepath"

	"github.com/rafaelescrich/gogents/internal/openrouter"
	"github.com/rafaelescrich/gogents/internal/rag"
	"github.com/rafaelescrich/gogents/internal/tools"
)

// Instance is a single agent with workspace, tools, and LLM config.
type Instance struct {
	ID            string
	Model         string
	Workspace     string
	MaxIterations int
	MaxTokens     int
	Temperature   float64
	Client        *openrouter.Client
	Tools         *tools.Registry
	Instructions  string
}

// Config for creating an agent instance.
type Config struct {
	Model         string
	Workspace     string
	MaxIterations int
	MaxTokens     int
	Temperature   float64
	OpenRouterKey string
	OpenRouterURL string
	Instructions  string
	// RAG
	RedVectorURL string
	EmbedAPIURL  string
	EmbedAPIKey  string
	EmbedModel   string
}

// NewInstance creates an agent instance and registers default tools.
func NewInstance(cfg Config) *Instance {
	if cfg.Workspace == "" {
		cfg.Workspace = "."
	}
	workspace, _ := filepath.Abs(cfg.Workspace)
	os.MkdirAll(workspace, 0755)

	if cfg.Model == "" {
		cfg.Model = "openrouter/auto"
	}
	if cfg.MaxIterations <= 0 {
		cfg.MaxIterations = 10
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = 8192
	}
	if cfg.Temperature == 0 {
		cfg.Temperature = 0.7
	}

	client := openrouter.NewClient(cfg.OpenRouterKey, cfg.OpenRouterURL)
	reg := tools.NewRegistry()

	// File tools (restrict to workspace)
	reg.Register(tools.NewReadFileTool(workspace, true))
	reg.Register(tools.NewWriteFileTool(workspace, true))
	reg.Register(tools.NewListDirTool(workspace, true))
	// Shell
	reg.Register(tools.NewShellTool(workspace, true, 0))
	// Web
	reg.Register(tools.NewWebFetchTool(0))

	// RAG (RedVector) - optional
	if cfg.RedVectorURL != "" {
		rv := rag.NewClient(cfg.RedVectorURL)
		var embedder tools.Embedder
		if cfg.EmbedAPIURL != "" {
			embedder = &embedAdapter{rag.NewEmbedClient(cfg.EmbedAPIKey, cfg.EmbedAPIURL, cfg.EmbedModel)}
		}
		reg.Register(tools.NewRAGSearchTool(rv, embedder))
	}

	instructions := cfg.Instructions
	if instructions == "" {
		instructions = defaultInstructions
	}

	return &Instance{
		ID:            "main",
		Model:         cfg.Model,
		Workspace:     workspace,
		MaxIterations: cfg.MaxIterations,
		MaxTokens:     cfg.MaxTokens,
		Temperature:   cfg.Temperature,
		Client:        client,
		Tools:         reg,
		Instructions:  instructions,
	}
}

const defaultInstructions = `You are a helpful AI assistant running locally. You can:
- Read, write, and list files in the workspace
- Run shell commands (with safety restrictions)
- Fetch web pages
- Search a RAG knowledge base (RedVector) when rag_search is available

Be concise and accurate. Prefer safe, reversible actions. When using run_shell, use simple commands and avoid destructive operations.`

// embedAdapter adapts rag.EmbedClient to tools.Embedder.
type embedAdapter struct{ *rag.EmbedClient }

func (e *embedAdapter) Embed(ctx context.Context, text string) ([]float64, error) {
	return e.EmbedClient.Embed(ctx, text)
}
