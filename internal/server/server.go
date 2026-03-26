// Package server exposes gogents as an OpenAI-compatible HTTP API for Cursor and other clients.
// All in Go: optional automatic HTTPS via Let's Encrypt (CertMagic).
package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/rafaelescrich/gogents/internal/agent"
)

// Server runs an HTTP server that exposes POST /v1/chat/completions (OpenAI-compat).
type Server struct {
	Agent       *agent.Instance
	Listen      string
	APIKey      string // optional; request must have Authorization: Bearer <APIKey>
	CertFile    string // optional manual TLS cert
	KeyFile     string // optional manual TLS key
	ServeDomain string // optional; if set, use Let's Encrypt (CertMagic) for HTTPS on :443
	ACMEEmail   string // required when ServeDomain is set (for Let's Encrypt)
}

// OpenAI-compat request (subset we need).
type chatRequest struct {
	Model    string          `json:"model"`
	Messages []chatMessage   `json:"messages"`
	Stream   bool            `json:"stream,omitempty"`
	MaxTokens *int           `json:"max_tokens,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAI-compat response (non-streaming).
type chatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// Run starts the HTTP server. Blocks until the server exits.
func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", s.auth(s.handleChatCompletions))
	mux.HandleFunc("/health", s.auth(s.handleHealth))

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
	}

	// Automatic HTTPS with Let's Encrypt (all in Go, no Caddy)
	if s.ServeDomain != "" {
		if s.ACMEEmail == "" {
			s.ACMEEmail = "admin@" + s.ServeDomain
		}
		certmagic.DefaultACME.Agreed = true
		certmagic.DefaultACME.Email = s.ACMEEmail
		cfg := certmagic.NewDefault()

		// HTTP-01 must be served on :80 during issuance and renewals (see CertMagic docs).
		am, ok := cfg.Issuers[0].(*certmagic.ACMEIssuer)
		if !ok {
			return fmt.Errorf("certmagic: expected ACME issuer")
		}
		httpToHTTPS := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := "https://" + s.ServeDomain + r.URL.RequestURI()
			http.Redirect(w, r, u, http.StatusMovedPermanently)
		})
		http80 := &http.Server{
			Handler:           am.HTTPChallengeHandler(httpToHTTPS),
			ReadHeaderTimeout: 15 * time.Second,
			ReadTimeout:       30 * time.Second,
		}
		ln80, err := net.Listen("tcp", ":80")
		if err != nil {
			return fmt.Errorf("listen :80 (needed for Let's Encrypt HTTP-01): %w", err)
		}
		go func() {
			if err := http80.Serve(ln80); err != nil && err != http.ErrServerClosed {
				log.Printf("http :80: %v", err)
			}
		}()
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_ = http80.Shutdown(shutdownCtx)
		}()

		if err := cfg.ManageSync(ctx, []string{s.ServeDomain}); err != nil {
			return err
		}
		tlsConfig := cfg.TLSConfig()
		srv.Addr = ":443"
		srv.TLSConfig = tlsConfig
		go func() {
			<-ctx.Done()
			srv.Shutdown(context.Background())
		}()
		log.Printf("gogents server HTTPS on https://%s (Let's Encrypt)", s.ServeDomain)
		ln, err := tls.Listen("tcp", ":443", tlsConfig)
		if err != nil {
			return err
		}
		return srv.Serve(ln)
	}

	srv.Addr = s.Listen
	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	if s.CertFile != "" && s.KeyFile != "" {
		log.Printf("gogents server HTTPS on %s", s.Listen)
		return srv.ListenAndServeTLS(s.CertFile, s.KeyFile)
	}
	log.Printf("gogents server HTTP on %s (use ngrok or TLS for Cursor)", s.Listen)
	return srv.ListenAndServe()
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.APIKey != "" {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != s.APIKey {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":{"message":"invalid or missing API key"}}`))
				return
			}
		}
		next(w, r)
	}
}

func (s *Server) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Last user (or assistant-then-user) content as the turn to run
	userContent := lastUserContent(req.Messages)
	if userContent == "" {
		writeError(w, "no user message in messages", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	out, err := s.Agent.Run(ctx, userContent)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	model := req.Model
	if model == "" {
		model = s.Agent.Model
	}
	resp := chatResponse{
		ID:      "gogents-" + time.Now().Format("20060102150405"),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}{
			{
				Index: 0,
				Message: struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				}{Role: "assistant", Content: out},
				FinishReason: "stop",
			},
		},
	}
	json.NewEncoder(w).Encode(resp)
}

func lastUserContent(messages []chatMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" && messages[i].Content != "" {
			return strings.TrimSpace(messages[i].Content)
		}
	}
	return ""
}

func writeError(w http.ResponseWriter, message string, code int) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{"message": message},
	})
}
