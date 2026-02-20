// Package openrouter provides an OpenRouter API client (OpenAI-compatible).
// See https://openrouter.ai/docs and https://openrouter.ai/docs/api-reference/chat-completion
package openrouter

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

const DefaultBaseURL = "https://openrouter.ai/api/v1"

// Client calls OpenRouter's chat completions API.
type Client struct {
	APIKey    string
	BaseURL   string
	HTTPClient *http.Client
}

// Message represents a chat message.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall from the model.
type ToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type,omitempty"`
	Function *FunctionCall   `json:"function,omitempty"`
	Name     string          `json:"name,omitempty"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// FunctionCall part of a tool call.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolDefinition for the API.
type ToolDefinition struct {
	Type     string              `json:"type"`
	Function ToolFunctionDef     `json:"function"`
}

// ToolFunctionDef describes a tool's name, description, and parameters.
type ToolFunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ChatRequest is the request body for POST /chat/completions.
type ChatRequest struct {
	Model       string            `json:"model"`
	Messages    []Message         `json:"messages"`
	Tools       []ToolDefinition  `json:"tools,omitempty"`
	ToolChoice  string            `json:"tool_choice,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature *float64          `json:"temperature,omitempty"`
	Stream      bool              `json:"stream,omitempty"`
}

// ChatResponse is the non-streaming response.
type ChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Index   int     `json:"index"`
		Message struct {
			Role       string     `json:"role"`
			Content    string     `json:"content"`
			ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// LLMResponse is the normalized response for the agent.
type LLMResponse struct {
	Content      string
	ToolCalls    []ToolCall
	FinishReason string
	Usage        *UsageInfo
}

// UsageInfo holds token counts.
type UsageInfo struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// NewClient creates an OpenRouter client.
func NewClient(apiKey, baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		APIKey:  apiKey,
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Chat sends a chat completion request and returns the response.
func (c *Client) Chat(ctx context.Context, messages []Message, tools []ToolDefinition, model string, maxTokens int, temperature float64) (*LLMResponse, error) {
	if model == "" {
		model = "openrouter/auto"
	}
	reqBody := ChatRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: &temperature,
		Stream:      false,
	}
	if len(tools) > 0 {
		reqBody.Tools = tools
		reqBody.ToolChoice = "auto"
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openrouter api %s: %s", resp.Status, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := chatResp.Choices[0]
	out := &LLMResponse{
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
		ToolCalls:    choice.Message.ToolCalls,
	}
	if chatResp.Usage != nil {
		out.Usage = &UsageInfo{
			PromptTokens:     chatResp.Usage.PromptTokens,
			CompletionTokens: chatResp.Usage.CompletionTokens,
			TotalTokens:      chatResp.Usage.TotalTokens,
		}
	}
	return out, nil
}
