package tools

import (
	"context"
	"testing"
)

func TestRegistry_RegisterGetList(t *testing.T) {
	r := NewRegistry()
	f := &fakeTool{name: "alpha", description: "First"}
	r.Register(f)
	if _, ok := r.Get("alpha"); !ok {
		t.Error("Get(alpha) want true")
	}
	if _, ok := r.Get("beta"); ok {
		t.Error("Get(beta) want false")
	}
	list := r.List()
	if len(list) != 1 || list[0] != "alpha" {
		t.Errorf("List() = %v", list)
	}
	r.Register(&fakeTool{name: "beta"})
	list = r.List()
	if len(list) != 2 {
		t.Errorf("List() len = %d", len(list))
	}
}

func TestRegistry_Execute_NotFound(t *testing.T) {
	r := NewRegistry()
	ctx := context.Background()
	res := r.Execute(ctx, "missing", nil)
	if !res.IsError {
		t.Error("Execute(missing) want error result")
	}
	if res.ForLLM == "" {
		t.Error("ForLLM want non-empty")
	}
}

func TestRegistry_Execute_Found(t *testing.T) {
	r := NewRegistry()
	r.Register(&fakeTool{name: "echo", result: OkResult("hello")})
	ctx := context.Background()
	res := r.Execute(ctx, "echo", map[string]interface{}{})
	if res.IsError {
		t.Error("Execute(echo) want success")
	}
	if res.ForLLM != "hello" {
		t.Errorf("ForLLM = %q", res.ForLLM)
	}
}

func TestRegistry_ToProviderDefs(t *testing.T) {
	r := NewRegistry()
	r.Register(&fakeTool{name: "a", description: "A tool"})
	r.Register(&fakeTool{name: "b", description: "B tool"})
	defs := r.ToProviderDefs()
	if len(defs) != 2 {
		t.Fatalf("ToProviderDefs() len = %d", len(defs))
	}
	names := make(map[string]bool)
	for _, d := range defs {
		if d.Type != "function" {
			t.Errorf("Type = %q", d.Type)
		}
		names[d.Function.Name] = true
	}
	if !names["a"] || !names["b"] {
		t.Errorf("names = %v", names)
	}
}
