package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const webUserAgent = "gogents/1.0"

// WebFetchTool fetches a URL and returns the response body (text).
type WebFetchTool struct {
	timeout time.Duration
}

// NewWebFetchTool creates a web_fetch tool.
func NewWebFetchTool(timeout time.Duration) *WebFetchTool {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &WebFetchTool{timeout: timeout}
}

func (t *WebFetchTool) Name() string        { return "web_fetch" }
func (t *WebFetchTool) Description() string { return "Fetch the content of a URL and return the response body as text. Use for reading web pages or APIs." }
func (t *WebFetchTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"url": map[string]interface{}{"type": "string", "description": "URL to fetch (e.g. https://example.com)"},
		},
		"required": []string{"url"},
	}
}

func (t *WebFetchTool) Execute(ctx context.Context, args map[string]interface{}) *Result {
	rawURL, _ := args["url"].(string)
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ErrorResult("url is required", nil)
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ErrorResult("invalid URL", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrorResult("only http and https are allowed", nil)
	}
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return ErrorResult(fmt.Sprintf("create request: %v", err), err)
	}
	req.Header.Set("User-Agent", webUserAgent)
	client := &http.Client{Timeout: t.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("request failed: %v", err), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ErrorResult(fmt.Sprintf("HTTP %s", resp.Status), nil)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return ErrorResult(fmt.Sprintf("read body: %v", err), err)
	}
	// Best-effort text; could add charset detection later
	return OkResult(string(body))
}
