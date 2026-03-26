// gogents is a local AI agent that uses OpenRouter models and RedVector RAG.
// Run: OPENROUTER_API_KEY=sk-or-... gogents
// Or: gogents "your question"
// Server (for Cursor / custom LLM): gogents --serve or GOGENTS_SERVE=1 gogents
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rafaelescrich/gogents/internal/agent"
	"github.com/rafaelescrich/gogents/internal/config"
	"github.com/rafaelescrich/gogents/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	// API key required only for cloud (OpenRouter); local (Ollama) ignores it
	if cfg.OpenRouterAPIKey == "" && !cfg.IsLocal() {
		fmt.Fprintln(os.Stderr, "Set OPENROUTER_API_KEY for cloud, or use a local backend: OLLAMA_HOST=localhost:11434 (or LLM_BASE_URL=http://localhost:11434/v1)")
		os.Exit(1)
	}

	agentCfg := agent.Config{
		Model:         cfg.Model,
		Workspace:     cfg.Workspace,
		MaxIterations: cfg.MaxIterations,
		MaxTokens:     cfg.MaxTokens,
		Temperature:   cfg.Temperature,
		OpenRouterKey: cfg.OpenRouterAPIKey,
		OpenRouterURL: cfg.OpenRouterURL,
		Instructions:  cfg.Instructions,
		RedVectorURL:  cfg.RedVectorURL,
		EmbedAPIURL:   cfg.EmbedAPIURL,
		EmbedAPIKey:   cfg.EmbedAPIKey,
		EmbedModel:    cfg.EmbedModel,
	}

	a := agent.NewInstance(agentCfg)

	// Server mode: OpenAI-compat API for Cursor / ngrok
	serveMode := os.Getenv("GOGENTS_SERVE") == "1" || os.Getenv("GOGENTS_SERVE") == "true"
	for _, arg := range os.Args[1:] {
		if arg == "--serve" || arg == "-s" {
			serveMode = true
			break
		}
	}
	if serveMode {
		listen := cfg.ServeAddr
		if listen == "" {
			listen = ":8080"
		}
		srv := &server.Server{
			Agent:        a,
			Listen:       listen,
			APIKey:       cfg.ServeAPIKey,
			CertFile:     cfg.ServeTLSCert,
			KeyFile:      cfg.ServeTLSKey,
			ServeDomain:  cfg.ServeDomain,
			ACMEEmail:    cfg.ServeACMEEmail,
		}
		if err := srv.Run(context.Background()); err != nil && err != context.Canceled {
			fmt.Fprintf(os.Stderr, "server: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if len(os.Args) > 1 {
		// One-shot: gogents "question"
		msg := strings.Join(os.Args[1:], " ")
		out, err := a.Run(context.Background(), msg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(out)
		return
	}

	// Interactive
	backend := "OpenRouter"
	if cfg.IsLocal() {
		backend = "Ollama/local"
	}
	fmt.Printf("gogents – agent (%s + RedVector RAG)\n", backend)
	fmt.Printf("model=%s workspace=%s\n", a.Model, a.Workspace)
	fmt.Println("Type a message and press Enter. Ctrl+D or /quit to exit.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "/quit" || line == "/exit" {
			break
		}
		out, err := a.Run(context.Background(), line)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			continue
		}
		fmt.Println(out)
		fmt.Println()
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "read: %v\n", err)
		os.Exit(1)
	}
}
