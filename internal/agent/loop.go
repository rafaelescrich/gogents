package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rafaelescrich/gogents/internal/openrouter"
)

// Loop runs the agent: send user message, call LLM, execute tool calls, repeat.
func (a *Instance) Run(ctx context.Context, userMessage string) (string, error) {
	messages := a.buildMessages(nil, userMessage)
	iteration := 0
	var finalContent string

	for iteration < a.MaxIterations {
		iteration++
		toolDefs := a.Tools.ToProviderDefs()

		resp, err := a.Client.Chat(ctx, messages, toolDefs, a.Model, a.MaxTokens, a.Temperature)
		if err != nil {
			return "", fmt.Errorf("llm call: %w", err)
		}

		finalContent = strings.TrimSpace(resp.Content)

		if len(resp.ToolCalls) == 0 {
			break
		}

		// Append assistant message with tool calls
		assistantMsg := openrouter.Message{
			Role:    "assistant",
			Content: resp.Content,
			ToolCalls: resp.ToolCalls,
		}
		messages = append(messages, assistantMsg)

		// Execute each tool call and append tool result
		for _, tc := range resp.ToolCalls {
			name := tc.Name
			if name == "" && tc.Function != nil {
				name = tc.Function.Name
			}
			args := tc.Arguments
			if args == nil && tc.Function != nil && tc.Function.Arguments != "" {
				var m map[string]interface{}
				if json.Unmarshal([]byte(tc.Function.Arguments), &m) == nil {
					args = m
				}
			}
			if args == nil {
				args = make(map[string]interface{})
			}

			result := a.Tools.Execute(ctx, name, args)
			contentForLLM := result.ForLLM
			if contentForLLM == "" && result.Err != nil {
				contentForLLM = result.Err.Error()
			}
			toolCallID := tc.ID
			if toolCallID == "" {
				toolCallID = "call_" + name
			}
			messages = append(messages, openrouter.Message{
				Role:       "tool",
				Content:    contentForLLM,
				ToolCallID: toolCallID,
			})
		}
	}

	return finalContent, nil
}

// buildMessages returns OpenRouter messages: system + history + user.
func (a *Instance) buildMessages(history []openrouter.Message, userContent string) []openrouter.Message {
	out := make([]openrouter.Message, 0, 2+len(history)+1)
	out = append(out, openrouter.Message{
		Role:    "system",
		Content: a.Instructions,
	})
	for _, m := range history {
		out = append(out, m)
	}
	out = append(out, openrouter.Message{
		Role:    "user",
		Content: userContent,
	})
	return out
}

// ToProviderDefs converts tool registry to OpenRouter tool definitions.
func (a *Instance) ToProviderDefs() []openrouter.ToolDefinition {
	return a.Tools.ToProviderDefs()
}
