package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rafaelescrich/gogents/internal/rag"
)

func TestRAGSearchTool_NameDescriptionParameters(t *testing.T) {
	rv := rag.NewClient("http://localhost:8888")
	tool := NewRAGSearchTool(rv, nil)
	if tool.Name() != "rag_search" {
		t.Errorf("Name() = %q", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("Description() empty")
	}
	params := tool.Parameters()
	if params["type"] != "object" {
		t.Errorf("Parameters() type = %v", params["type"])
	}
}

func TestRAGSearchTool_NoEmbedder(t *testing.T) {
	rv := rag.NewClient("http://localhost:8888")
	tool := NewRAGSearchTool(rv, nil)
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{
		"collection": "docs",
		"query":      "test",
	})
	if !res.IsError {
		t.Error("want error when embedder is nil")
	}
}

func TestRAGSearchTool_MissingCollection(t *testing.T) {
	rv := rag.NewClient("http://localhost:8888")
	tool := NewRAGSearchTool(rv, &mockEmbedder{vec: []float64{0.1}})
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{"query": "test"})
	if !res.IsError {
		t.Error("want error for missing collection")
	}
}

func TestRAGSearchTool_MissingQuery(t *testing.T) {
	rv := rag.NewClient("http://localhost:8888")
	tool := NewRAGSearchTool(rv, &mockEmbedder{vec: []float64{0.1}})
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{"collection": "docs"})
	if !res.IsError {
		t.Error("want error for missing query")
	}
}

type mockEmbedder struct {
	vec []float64
	err error
}

func (m *mockEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.vec, nil
}

func TestRAGSearchTool_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/collections/docs/search" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result": []map[string]interface{}{
				{"id": "1", "score": 0.95, "payload": map[string]interface{}{"text": "chunk one"}},
				{"id": "2", "score": 0.8, "payload": map[string]interface{}{"content": "chunk two"}},
			},
		})
	}))
	defer server.Close()

	rv := rag.NewClient(server.URL)
	tool := NewRAGSearchTool(rv, &mockEmbedder{vec: []float64{0.1, 0.2}})
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{
		"collection": "docs",
		"query":      "test",
		"limit":      float64(5),
	})
	if res.IsError {
		t.Fatalf("Execute: %v", res.Err)
	}
	if res.ForLLM == "" {
		t.Error("ForLLM empty")
	}
	if res.ForLLM == "No results found for the query." {
		t.Error("expected results from mock server")
	}
}

func TestRAGSearchTool_NoResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"result": []interface{}{}})
	}))
	defer server.Close()

	rv := rag.NewClient(server.URL)
	tool := NewRAGSearchTool(rv, &mockEmbedder{vec: []float64{0.1}})
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{
		"collection": "empty",
		"query":      "test",
	})
	if res.IsError {
		t.Fatalf("Execute: %v", res.Err)
	}
	if res.ForLLM != "No results found for the query." {
		t.Errorf("ForLLM = %q", res.ForLLM)
	}
}

func TestRAGSearchTool_EmbedderError(t *testing.T) {
	rv := rag.NewClient("http://localhost:8888")
	tool := NewRAGSearchTool(rv, &mockEmbedder{err: context.DeadlineExceeded})
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{
		"collection": "docs",
		"query":      "test",
	})
	if !res.IsError {
		t.Error("want error when embedder fails")
	}
}
