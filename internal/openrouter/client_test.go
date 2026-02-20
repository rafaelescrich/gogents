package openrouter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClient_DefaultBaseURL(t *testing.T) {
	c := NewClient("key", "")
	if c.BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, DefaultBaseURL)
	}
	if c.APIKey != "key" {
		t.Errorf("APIKey = %q", c.APIKey)
	}
}

func TestNewClient_TrimSlash(t *testing.T) {
	c := NewClient("", "https://api.example.com/v1/")
	if c.BaseURL != "https://api.example.com/v1" {
		t.Errorf("BaseURL = %q", c.BaseURL)
	}
}

func TestClient_Chat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" || r.Method != "POST" {
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
						"content": "Hello!",
					},
					"finish_reason": "stop",
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL)
	ctx := context.Background()
	messages := []Message{{Role: "user", Content: "Hi"}}
	resp, err := client.Chat(ctx, messages, nil, "test-model", 100, 0.7)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "Hello!" {
		t.Errorf("Content = %q", resp.Content)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q", resp.FinishReason)
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("ToolCalls = %v", resp.ToolCalls)
	}
}

func TestClient_Chat_WithToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
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
									"name":      "get_time",
									"arguments": `{"timezone":"UTC"}`,
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient("key", server.URL)
	ctx := context.Background()
	resp, err := client.Chat(ctx, []Message{{Role: "user", Content: "Time?"}}, nil, "m", 100, 0)
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("ToolCalls len = %d", len(resp.ToolCalls))
	}
	tc := resp.ToolCalls[0]
	if tc.ID != "call_1" {
		t.Errorf("ToolCall ID = %q", tc.ID)
	}
	if tc.Function == nil {
		t.Fatal("Function nil")
	}
	if tc.Function.Name != "get_time" {
		t.Errorf("Function.Name = %q", tc.Function.Name)
	}
	if tc.Function.Arguments != `{"timezone":"UTC"}` {
		t.Errorf("Function.Arguments = %q", tc.Function.Arguments)
	}
}

func TestClient_Chat_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("invalid key"))
	}))
	defer server.Close()

	client := NewClient("bad", server.URL)
	ctx := context.Background()
	_, err := client.Chat(ctx, nil, nil, "m", 100, 0)
	if err == nil {
		t.Fatal("Chat want error")
	}
}

func TestClient_Chat_NoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"choices": []interface{}{}})
	}))
	defer server.Close()

	client := NewClient("key", server.URL)
	ctx := context.Background()
	_, err := client.Chat(ctx, nil, nil, "m", 100, 0)
	if err == nil {
		t.Fatal("Chat want error when no choices")
	}
}
