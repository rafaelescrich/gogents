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

// EmbedClient calls an OpenAI-compatible embeddings API.
// OpenRouter supports embeddings; or use OpenAI / local embed model.
type EmbedClient struct {
	APIKey     string
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

// NewEmbedClient creates an embed client. baseURL is e.g. https://openrouter.ai/api/v1 or https://api.openai.com/v1
func NewEmbedClient(apiKey, baseURL, model string) *EmbedClient {
	baseURL = strings.TrimRight(baseURL, "/")
	if model == "" {
		model = "openai/text-embedding-3-small"
	}
	return &EmbedClient{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// embedRequest for POST .../embeddings
type embedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

// embedResponse from embeddings API
type embedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

// Embed returns the embedding vector for the given text.
func (c *EmbedClient) Embed(ctx context.Context, text string) ([]float64, error) {
	body := embedRequest{Model: c.Model, Input: text}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/embeddings", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
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
		return nil, fmt.Errorf("embeddings API %s: %s", resp.Status, string(data))
	}
	var out embedResponse
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if len(out.Data) == 0 || len(out.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding in response")
	}
	return out.Data[0].Embedding, nil
}
