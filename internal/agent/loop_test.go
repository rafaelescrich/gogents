package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestInstance_Run_DirectResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "chat-1",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Hello, I am the assistant.",
					},
					"finish_reason": "stop",
				},
			},
		})
	}))
	defer server.Close()

	cfg := Config{
		Model:         "test",
		Workspace:     t.TempDir(),
		MaxIterations: 5,
		MaxTokens:     100,
		Temperature:   0.7,
		OpenRouterKey: "key",
		OpenRouterURL: server.URL,
		Instructions:  "You are helpful.",
	}
	a := NewInstance(cfg)
	ctx := context.Background()

	out, err := a.Run(ctx, "Hi")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out != "Hello, I am the assistant." {
		t.Errorf("out = %q", out)
	}
}

func TestInstance_Run_WithToolCall(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			// First response: model asks to run read_file
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "chat-1",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": "",
							"tool_calls": []map[string]interface{}{
								{
									"id":   "call_1",
									"type": "function",
									"function": map[string]interface{}{
										"name":      "read_file",
										"arguments": `{"path": "test.txt"}`,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
			})
			return
		}
		// Second response: final answer
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "chat-2",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "The file contains: hello world",
					},
					"finish_reason": "stop",
				},
			},
		})
	}))
	defer server.Close()

	dir := t.TempDir()
	if err := writeFile(dir+"/test.txt", "hello world"); err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		Model:         "test",
		Workspace:     dir,
		MaxIterations: 5,
		MaxTokens:     100,
		Temperature:   0.7,
		OpenRouterKey: "key",
		OpenRouterURL: server.URL,
		Instructions:  "You are helpful.",
	}
	a := NewInstance(cfg)
	ctx := context.Background()

	out, err := a.Run(ctx, "What is in test.txt?")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out != "The file contains: hello world" {
		t.Errorf("out = %q", out)
	}
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2", callCount)
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// TestInstance_Run_WithRAGAndEmbedder covers embedAdapter.Embed by running a turn that calls rag_search with embedder.
func TestInstance_Run_WithRAGAndEmbedder(t *testing.T) {
	// Embed server: returns a small vector so RedVector search can run
	embedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"embedding": []float64{0.1, 0.2, 0.3}},
			},
		})
	}))
	defer embedSrv.Close()

	// RedVector server: search returns one result
	rvSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/collections/docs/search" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result": []map[string]interface{}{
				{"id": "1", "score": 0.9, "payload": map[string]interface{}{"text": "RAG result content"}},
			},
		})
	}))
	defer rvSrv.Close()

	callCount := 0
	chatSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "chat-1",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": "",
							"tool_calls": []map[string]interface{}{
								{
									"id":   "call_rag",
									"type": "function",
									"function": map[string]interface{}{
										"name":      "rag_search",
										"arguments": `{"collection":"docs","query":"test"}`,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "chat-2",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Found: RAG result content",
					},
					"finish_reason": "stop",
				},
			},
		})
	}))
	defer chatSrv.Close()

	cfg := Config{
		Model:         "test",
		Workspace:     t.TempDir(),
		MaxIterations: 5,
		MaxTokens:     100,
		Temperature:   0.7,
		OpenRouterKey: "key",
		OpenRouterURL: chatSrv.URL,
		RedVectorURL:  rvSrv.URL,
		EmbedAPIURL:   embedSrv.URL,
		EmbedModel:    "test-embed",
		Instructions:  "You are helpful.",
	}
	a := NewInstance(cfg)
	ctx := context.Background()

	out, err := a.Run(ctx, "Search docs for test")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out != "Found: RAG result content" {
		t.Errorf("out = %q", out)
	}
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2", callCount)
	}
}
