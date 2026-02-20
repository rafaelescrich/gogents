package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rafaelescrich/gogents/internal/rag"
)

// Embedder produces a vector from a text (e.g. for RAG query).
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

// RAGSearchTool searches a RedVector collection by query text.
// If no Embedder is set, the tool returns an error asking to configure one.
type RAGSearchTool struct {
	redvector *rag.Client
	embedder  Embedder
}

// NewRAGSearchTool creates a rag_search tool using RedVector and an optional embedder.
func NewRAGSearchTool(redvector *rag.Client, embedder Embedder) *RAGSearchTool {
	return &RAGSearchTool{redvector: redvector, embedder: embedder}
}

func (t *RAGSearchTool) Name() string        { return "rag_search" }
func (t *RAGSearchTool) Description() string { return "Search a RedVector RAG collection by natural language query. Returns the most relevant stored chunks. Requires the collection to exist and an embedding API to be configured for query embedding." }
func (t *RAGSearchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"collection": map[string]interface{}{"type": "string", "description": "RedVector collection name"},
			"query":      map[string]interface{}{"type": "string", "description": "Natural language search query"},
			"limit":      map[string]interface{}{"type": "integer", "description": "Max number of results (default 5)"},
		},
		"required": []string{"collection", "query"},
	}
}

func (t *RAGSearchTool) Execute(ctx context.Context, args map[string]interface{}) *Result {
	collection, _ := args["collection"].(string)
	query, _ := args["query"].(string)
	limit, _ := args["limit"].(float64)
	collection = strings.TrimSpace(collection)
	query = strings.TrimSpace(query)
	if collection == "" || query == "" {
		return ErrorResult("collection and query are required", nil)
	}
	if t.embedder == nil {
		return ErrorResult("RAG search requires an embedding API to be configured (embedder). Set EMBED_API_URL and EMBED_API_KEY for query embedding.", nil)
	}
	vec, err := t.embedder.Embed(ctx, query)
	if err != nil {
		return ErrorResult(fmt.Sprintf("embed query: %v", err), err)
	}
	if len(vec) == 0 {
		return ErrorResult("embedder returned empty vector", nil)
	}
	lim := uint64(5)
	if limit > 0 && limit <= 20 {
		lim = uint64(limit)
	}
	results, err := t.redvector.Search(ctx, collection, vec, lim)
	if err != nil {
		return ErrorResult(fmt.Sprintf("redvector search: %v", err), err)
	}
	if len(results) == 0 {
		return OkResult("No results found for the query.")
	}
	var sb strings.Builder
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("[%d] score=%.4f id=%v\n", i+1, r.Score, r.ID))
		if r.Payload != nil {
			if text, ok := r.Payload["text"].(string); ok {
				sb.WriteString(text)
				sb.WriteString("\n")
			} else if content, ok := r.Payload["content"].(string); ok {
				sb.WriteString(content)
				sb.WriteString("\n")
			} else {
				j, _ := json.Marshal(r.Payload)
				sb.WriteString(string(j))
				sb.WriteString("\n")
			}
		}
	}
	return OkResult(sb.String())
}
