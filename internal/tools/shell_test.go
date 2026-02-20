package tools

import (
	"context"
	"path/filepath"
	"testing"
)

func TestShellTool_NameDescriptionParameters(t *testing.T) {
	tool := NewShellTool("/tmp", true, 0)
	if tool.Name() != "run_shell" {
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

func TestShellTool_MissingCommand(t *testing.T) {
	dir := t.TempDir()
	tool := NewShellTool(dir, true, 0)
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{})
	if !res.IsError {
		t.Error("want error for missing command")
	}
}

func TestShellTool_EmptyCommand(t *testing.T) {
	dir := t.TempDir()
	tool := NewShellTool(dir, true, 0)
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{"command": "   "})
	if !res.IsError {
		t.Error("want error for empty command")
	}
}

func TestShellTool_DangerousCommand(t *testing.T) {
	dir := t.TempDir()
	tool := NewShellTool(dir, true, 0)
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{"command": "rm -rf /"})
	if !res.IsError {
		t.Error("want error for dangerous command")
	}
}

func TestShellTool_Success(t *testing.T) {
	dir := t.TempDir()
	tool := NewShellTool(dir, true, 0)
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{"command": "echo hello"})
	if res.IsError {
		t.Fatalf("Execute: %v", res.Err)
	}
	if res.ForLLM != "hello" {
		t.Errorf("ForLLM = %q", res.ForLLM)
	}
}

func TestShellTool_WorkingDir(t *testing.T) {
	dir := t.TempDir()
	tool := NewShellTool(dir, true, 0)
	ctx := context.Background()
	// pwd should print the workspace dir (or its resolved path)
	res := tool.Execute(ctx, map[string]interface{}{"command": "pwd"})
	if res.IsError {
		t.Fatalf("Execute: %v", res.Err)
	}
	abs, _ := filepath.Abs(dir)
	if res.ForLLM != abs {
		t.Errorf("ForLLM (pwd) = %q, want %q", res.ForLLM, abs)
	}
}
