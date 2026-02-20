package rag

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient_TrimSlash(t *testing.T) {
	c := NewClient("http://localhost:8888/")
	if c.BaseURL != "http://localhost:8888" {
		t.Errorf("BaseURL = %q", c.BaseURL)
	}
}

func TestClient_Search_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/collections/docs/search" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SearchResponse{
			Result: []SearchResultItem{
				{ID: "1", Score: 0.95, Payload: map[string]interface{}{"text": "chunk one"}},
				{ID: "2", Score: 0.8, Payload: map[string]interface{}{"content": "chunk two"}},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()
	results, err := client.Search(ctx, "docs", []float64{0.1, 0.2}, 5)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d", len(results))
	}
	if results[0].Score != 0.95 || results[0].ID != "1" {
		t.Errorf("results[0] = %+v", results[0])
	}
}

func TestClient_Search_DefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SearchResponse{Result: nil})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()
	_, err := client.Search(ctx, "c", []float64{0.1}, 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
}

func TestClient_Search_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()
	_, err := client.Search(ctx, "c", []float64{0.1}, 10)
	if err == nil {
		t.Fatal("Search want error")
	}
}

func TestClient_GetCollection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/collections/mycol" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result": map[string]interface{}{
				"status": "green",
				"config": map[string]interface{}{
					"params": map[string]interface{}{
						"vectors": map[string]interface{}{
							"size":     1024,
							"distance": "Cosine",
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()
	info, err := client.GetCollection(ctx, "mycol")
	if err != nil {
		t.Fatalf("GetCollection: %v", err)
	}
	if info.Result == nil || info.Result.Status != "green" {
		t.Errorf("Result = %+v", info.Result)
	}
}

func TestClient_ListCollections(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/collections" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result": map[string]interface{}{
				"collections": []map[string]interface{}{
					{"name": "a"},
					{"name": "b"},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()
	names, err := client.ListCollections(ctx)
	if err != nil {
		t.Fatalf("ListCollections: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("len(names) = %d", len(names))
	}
	if names[0] != "a" || names[1] != "b" {
		t.Errorf("names = %v", names)
	}
}
