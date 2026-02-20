package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebFetchTool_NameDescriptionParameters(t *testing.T) {
	tool := NewWebFetchTool(0)
	if tool.Name() != "web_fetch" {
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

func TestWebFetchTool_MissingURL(t *testing.T) {
	tool := NewWebFetchTool(0)
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{})
	if !res.IsError {
		t.Error("want error for missing url")
	}
	if res.ForLLM != "url is required" {
		t.Errorf("ForLLM = %q", res.ForLLM)
	}
}

func TestWebFetchTool_EmptyURL(t *testing.T) {
	tool := NewWebFetchTool(0)
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{"url": "   "})
	if !res.IsError {
		t.Error("want error for empty url")
	}
}

func TestWebFetchTool_InvalidURL(t *testing.T) {
	tool := NewWebFetchTool(0)
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{"url": "not-a-url"})
	if !res.IsError {
		t.Error("want error for invalid url")
	}
}

func TestWebFetchTool_UnsupportedScheme(t *testing.T) {
	tool := NewWebFetchTool(0)
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{"url": "ftp://example.com/file"})
	if !res.IsError {
		t.Error("want error for ftp")
	}
	if res.ForLLM != "only http and https are allowed" {
		t.Errorf("ForLLM = %q", res.ForLLM)
	}
}

func TestWebFetchTool_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello from server"))
	}))
	defer server.Close()

	tool := NewWebFetchTool(0)
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{"url": server.URL})
	if res.IsError {
		t.Fatalf("Execute: %v", res.Err)
	}
	if res.ForLLM != "hello from server" {
		t.Errorf("ForLLM = %q", res.ForLLM)
	}
}

func TestWebFetchTool_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tool := NewWebFetchTool(0)
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{"url": server.URL})
	if !res.IsError {
		t.Error("want error for 404")
	}
}
