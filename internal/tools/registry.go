package tools

import (
	"context"
	"sync"

	"github.com/rafaelescrich/gogents/internal/openrouter"
)

// Registry holds tools and can execute them and produce provider definitions.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds a tool.
func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

// Get returns a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List returns all tool names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools))
	for n := range r.tools {
		names = append(names, n)
	}
	return names
}

// Execute runs a tool by name with the given arguments.
func (r *Registry) Execute(ctx context.Context, name string, args map[string]interface{}) *Result {
	t, ok := r.Get(name)
	if !ok {
		return ErrorResult("tool not found: "+name, nil)
	}
	return t.Execute(ctx, args)
}

// ToProviderDefs returns OpenRouter tool definitions for the API.
func (r *Registry) ToProviderDefs() []openrouter.ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]openrouter.ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, openrouter.ToolDefinition{
			Type: "function",
			Function: openrouter.ToolFunctionDef{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		})
	}
	return defs
}
