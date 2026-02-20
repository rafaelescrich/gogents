package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadFileTool reads file contents from the workspace.
type ReadFileTool struct {
	workspace string
	restrict  bool
}

// NewReadFileTool creates a read_file tool.
func NewReadFileTool(workspace string, restrict bool) *ReadFileTool {
	return &ReadFileTool{workspace: workspace, restrict: restrict}
}

func (t *ReadFileTool) Name() string        { return "read_file" }
func (t *ReadFileTool) Description() string { return "Read the contents of a file. Path is relative to workspace unless absolute." }
func (t *ReadFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Relative or absolute path to the file",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, args map[string]interface{}) *Result {
	path, _ := args["path"].(string)
	if path == "" {
		return ErrorResult("path is required", nil)
	}
	resolved, err := validatePath(path, t.workspace, t.restrict)
	if err != nil {
		return ErrorResult(err.Error(), err)
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return ErrorResult(fmt.Sprintf("read file: %v", err), err)
	}
	return OkResult(string(data))
}

// WriteFileTool writes content to a file.
type WriteFileTool struct {
	workspace string
	restrict  bool
}

// NewWriteFileTool creates a write_file tool.
func NewWriteFileTool(workspace string, restrict bool) *WriteFileTool {
	return &WriteFileTool{workspace: workspace, restrict: restrict}
}

func (t *WriteFileTool) Name() string        { return "write_file" }
func (t *WriteFileTool) Description() string { return "Write content to a file. Creates parent dirs if needed. Path is relative to workspace unless absolute." }
func (t *WriteFileTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path":    map[string]interface{}{"type": "string", "description": "Relative or absolute path"},
			"content": map[string]interface{}{"type": "string", "description": "Content to write"},
		},
		"required": []string{"path", "content"},
	}
}

func (t *WriteFileTool) Execute(ctx context.Context, args map[string]interface{}) *Result {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	if path == "" {
		return ErrorResult("path is required", nil)
	}
	resolved, err := validatePath(path, t.workspace, t.restrict)
	if err != nil {
		return ErrorResult(err.Error(), err)
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0755); err != nil {
		return ErrorResult(fmt.Sprintf("mkdir: %v", err), err)
	}
	if err := os.WriteFile(resolved, []byte(content), 0644); err != nil {
		return ErrorResult(fmt.Sprintf("write: %v", err), err)
	}
	return OkResult("File written successfully.")
}

// ListDirTool lists directory contents.
type ListDirTool struct {
	workspace string
	restrict  bool
}

// NewListDirTool creates a list_dir tool.
func NewListDirTool(workspace string, restrict bool) *ListDirTool {
	return &ListDirTool{workspace: workspace, restrict: restrict}
}

func (t *ListDirTool) Name() string        { return "list_dir" }
func (t *ListDirTool) Description() string { return "List files and directories at the given path. Path is relative to workspace unless absolute." }
func (t *ListDirTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{"type": "string", "description": "Directory path (default: workspace root)"},
		},
	}
}

func (t *ListDirTool) Execute(ctx context.Context, args map[string]interface{}) *Result {
	path, _ := args["path"].(string)
	if path == "" {
		path = t.workspace
	}
	resolved, err := validatePath(path, t.workspace, t.restrict)
	if err != nil {
		return ErrorResult(err.Error(), err)
	}
	entries, err := os.ReadDir(resolved)
	if err != nil {
		return ErrorResult(fmt.Sprintf("list dir: %v", err), err)
	}
	var names []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			names = append(names, name+"/")
		} else {
			names = append(names, name)
		}
	}
	return OkResult(strings.Join(names, "\n"))
}

func validatePath(path, workspace string, restrict bool) (string, error) {
	if workspace == "" {
		workspace = "."
	}
	absWorkspace, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("workspace path: %w", err)
	}
	var absPath string
	if filepath.IsAbs(path) {
		absPath = filepath.Clean(path)
	} else {
		absPath, err = filepath.Abs(filepath.Join(absWorkspace, path))
		if err != nil {
			return "", fmt.Errorf("resolve path: %w", err)
		}
	}
	if restrict {
		rel, err := filepath.Rel(filepath.Clean(absWorkspace), filepath.Clean(absPath))
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return "", fmt.Errorf("access denied: path outside workspace")
		}
	}
	return absPath, nil
}
