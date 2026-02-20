package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestReadFileTool_Success(t *testing.T) {
	dir := t.TempDir()
	fpath := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(fpath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	tool := NewReadFileTool(dir, true)
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{"path": "f.txt"})
	if res.IsError {
		t.Fatalf("Execute: %v", res.Err)
	}
	if res.ForLLM != "hello" {
		t.Errorf("ForLLM = %q", res.ForLLM)
	}
}

func TestReadFileTool_MissingPath(t *testing.T) {
	tool := NewReadFileTool(t.TempDir(), true)
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{})
	if !res.IsError {
		t.Error("want error for missing path")
	}
}

func TestReadFileTool_NotFound(t *testing.T) {
	tool := NewReadFileTool(t.TempDir(), true)
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{"path": "nonexistent.txt"})
	if !res.IsError {
		t.Error("want error for nonexistent file")
	}
}

func TestReadFileTool_OutsideWorkspace(t *testing.T) {
	dir := t.TempDir()
	tool := NewReadFileTool(dir, true)
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{"path": "../../etc/passwd"})
	if !res.IsError {
		t.Error("want error for path outside workspace")
	}
}

func TestWriteFileTool_Success(t *testing.T) {
	dir := t.TempDir()
	tool := NewWriteFileTool(dir, true)
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{
		"path":    "subdir/f.txt",
		"content": "written",
	})
	if res.IsError {
		t.Fatalf("Execute: %v", res.Err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "subdir", "f.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "written" {
		t.Errorf("file content = %q", data)
	}
}

func TestWriteFileTool_MissingPath(t *testing.T) {
	tool := NewWriteFileTool(t.TempDir(), true)
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{"content": "x"})
	if !res.IsError {
		t.Error("want error for missing path")
	}
}

func TestListDirTool_Success(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a"), nil, 0644)
	os.MkdirAll(filepath.Join(dir, "d"), 0755)
	tool := NewListDirTool(dir, true)
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{"path": "."})
	if res.IsError {
		t.Fatalf("Execute: %v", res.Err)
	}
	if res.ForLLM == "" {
		t.Error("ForLLM want non-empty")
	}
}

func TestListDirTool_DefaultPath(t *testing.T) {
	dir := t.TempDir()
	tool := NewListDirTool(dir, true)
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{})
	if res.IsError {
		t.Fatalf("Execute: %v", res.Err)
	}
}

func TestListDirTool_NotFound(t *testing.T) {
	dir := t.TempDir()
	tool := NewListDirTool(dir, true)
	ctx := context.Background()
	res := tool.Execute(ctx, map[string]interface{}{"path": "nonexistent"})
	if !res.IsError {
		t.Error("want error for nonexistent dir")
	}
}
