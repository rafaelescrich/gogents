// Package rag provides a RedVector REST client for RAG (Qdrant-compatible API).
// See https://github.com/rafaelescrich/redvector
package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is a RedVector REST API client (default port 8888).
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a RedVector REST client. baseURL is e.g. http://localhost:8888
func NewClient(baseURL string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SearchRequest for POST /api/collections/:name/search
type SearchRequest struct {
	Vector []float64 `json:"vector"`
	Limit  uint64    `json:"limit,omitempty"`
}

// SearchResultItem is one result from search.
type SearchResultItem struct {
	ID      interface{}            `json:"id"`
	Score   float64                `json:"score"`
	Payload map[string]interface{} `json:"payload,omitempty"`
	Vector  []float64              `json:"vector,omitempty"`
}

// SearchResponse from RedVector search.
type SearchResponse struct {
	Result []SearchResultItem `json:"result"`
}

// Search runs a vector search on the given collection.
// vector must match the collection's dimension; limit is max results (default 10).
func (c *Client) Search(ctx context.Context, collection string, vector []float64, limit uint64) ([]SearchResultItem, error) {
	if limit == 0 {
		limit = 10
	}
	body := SearchRequest{Vector: vector, Limit: limit}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal search request: %w", err)
	}

	url := c.BaseURL + "/api/collections/" + collection + "/search"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("redvector search %s: %s", resp.Status, string(data))
	}

	var out SearchResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}
	return out.Result, nil
}

// CollectionInfo from GET /api/collections/:name
type CollectionInfo struct {
	Result *struct {
		Status string `json:"status"`
		Config *struct {
			Params *struct {
				Vectors *struct {
					Size     int    `json:"size"`
					Distance string `json:"distance"`
				} `json:"vectors"`
			} `json:"params"`
		} `json:"config"`
	} `json:"result"`
}

// GetCollection returns collection info (including vector size).
func (c *Client) GetCollection(ctx context.Context, collection string) (*CollectionInfo, error) {
	url := c.BaseURL + "/api/collections/" + collection
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("redvector get collection %s: %s", resp.Status, string(data))
	}
	var info CollectionInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// ListCollections returns collection names from GET /api/collections
type ListCollectionsResponse struct {
	Result *struct {
		Collections []struct {
			Name string `json:"name"`
		} `json:"collections"`
	} `json:"result"`
}

// ListCollections lists all collections.
func (c *Client) ListCollections(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/api/collections", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("redvector list collections %s: %s", resp.Status, string(data))
	}
	var out ListCollectionsResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if out.Result == nil {
		return nil, nil
	}
	names := make([]string, 0, len(out.Result.Collections))
	for _, c := range out.Result.Collections {
		names = append(names, c.Name)
	}
	return names, nil
}
