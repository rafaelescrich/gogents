package tools

import (
	"context"
	"fmt"
)

// Tool is the interface all tools implement.
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(ctx context.Context, args map[string]interface{}) *Result
}

// Result of a tool execution.
type Result struct {
	ForLLM  string // Content to send back to the LLM
	ForUser string // Optional user-facing message
	Err     error
	IsError bool
	Async   bool
	Silent  bool
}

// OkResult returns a successful result for the LLM.
func OkResult(forLLM string) *Result {
	return &Result{ForLLM: forLLM}
}

// ErrorResult returns an error result for the LLM.
func ErrorResult(msg string, err error) *Result {
	if err == nil {
		err = fmt.Errorf("%s", msg)
	}
	return &Result{ForLLM: msg, Err: err, IsError: true}
}

// ToSchema builds the OpenAI-style function schema for a tool.
func ToSchema(t Tool) map[string]interface{} {
	return map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        t.Name(),
			"description": t.Description(),
			"parameters":  t.Parameters(),
		},
	}
}
