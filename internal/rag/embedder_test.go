package rag

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewEmbedClient_DefaultModel(t *testing.T) {
	c := NewEmbedClient("key", "https://api.example.com/v1", "")
	if c.Model != "openai/text-embedding-3-small" {
		t.Errorf("Model = %q", c.Model)
	}
	if c.BaseURL != "https://api.example.com/v1" {
		t.Errorf("BaseURL = %q", c.BaseURL)
	}
}

func TestEmbedClient_Embed_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" || r.Method != "POST" {
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
	defer server.Close()

	client := NewEmbedClient("key", server.URL, "test-model")
	ctx := context.Background()
	vec, err := client.Embed(ctx, "hello")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vec) != 3 {
		t.Fatalf("len(vec) = %d", len(vec))
	}
	if vec[0] != 0.1 || vec[1] != 0.2 || vec[2] != 0.3 {
		t.Errorf("vec = %v", vec)
	}
}

func TestEmbedClient_Embed_NoKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"embedding": []float64{0.5}},
			},
		})
	}))
	defer server.Close()

	client := NewEmbedClient("", server.URL, "local")
	ctx := context.Background()
	vec, err := client.Embed(ctx, "hi")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vec) != 1 || vec[0] != 0.5 {
		t.Errorf("vec = %v", vec)
	}
}

func TestEmbedClient_Embed_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}))
	defer server.Close()

	client := NewEmbedClient("key", server.URL, "m")
	ctx := context.Background()
	_, err := client.Embed(ctx, "hi")
	if err == nil {
		t.Fatal("Embed want error")
	}
}

func TestEmbedClient_Embed_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}})
	}))
	defer server.Close()

	client := NewEmbedClient("key", server.URL, "m")
	ctx := context.Background()
	_, err := client.Embed(ctx, "hi")
	if err == nil {
		t.Fatal("Embed want error for empty data")
	}
}
