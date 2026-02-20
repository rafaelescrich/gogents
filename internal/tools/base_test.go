package tools

import (
	"context"
	"testing"
)

// fakeTool implements Tool for tests.
type fakeTool struct {
	name        string
	description string
	params      map[string]interface{}
	result      *Result
}

func (f *fakeTool) Name() string        { return f.name }
func (f *fakeTool) Description() string { return f.description }
func (f *fakeTool) Parameters() map[string]interface{} {
	if f.params != nil {
		return f.params
	}
	return map[string]interface{}{"type": "object"}
}
func (f *fakeTool) Execute(ctx context.Context, args map[string]interface{}) *Result {
	if f.result != nil {
		return f.result
	}
	return OkResult("ok")
}

func TestOkResult(t *testing.T) {
	r := OkResult("hello")
	if r.ForLLM != "hello" {
		t.Errorf("ForLLM = %q", r.ForLLM)
	}
	if r.IsError || r.Err != nil {
		t.Errorf("IsError=%v Err=%v", r.IsError, r.Err)
	}
}

func TestErrorResult(t *testing.T) {
	r := ErrorResult("failed", nil)
	if r.ForLLM != "failed" {
		t.Errorf("ForLLM = %q", r.ForLLM)
	}
	if !r.IsError {
		t.Error("IsError want true")
	}
	if r.Err == nil {
		t.Error("Err want non-nil")
	}
}

func TestErrorResult_WithErr(t *testing.T) {
	err := context.Canceled
	r := ErrorResult("canceled", err)
	if r.Err != err {
		t.Errorf("Err = %v", r.Err)
	}
}

func TestToSchema(t *testing.T) {
	f := &fakeTool{
		name:        "my_tool",
		description: "Does something",
		params: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{"x": "string"},
		},
	}
	s := ToSchema(f)
	if s["type"] != "function" {
		t.Errorf("type = %v", s["type"])
	}
	fn, ok := s["function"].(map[string]interface{})
	if !ok {
		t.Fatal("function not map")
	}
	if fn["name"] != "my_tool" {
		t.Errorf("name = %v", fn["name"])
	}
	if fn["description"] != "Does something" {
		t.Errorf("description = %v", fn["description"])
	}
}
