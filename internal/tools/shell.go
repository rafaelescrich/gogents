package tools

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ShellTool runs shell commands in the workspace with basic safety checks.
type ShellTool struct {
	workspace string
	restrict  bool
	timeout   time.Duration
	denyList  []*regexp.Regexp
}

// NewShellTool creates a run_shell tool.
func NewShellTool(workspace string, restrict bool, timeout time.Duration) *ShellTool {
	return &ShellTool{
		workspace: workspace,
		restrict:  restrict,
		timeout:   timeout,
		denyList:  defaultShellDenyList(),
	}
}

func defaultShellDenyList() []*regexp.Regexp {
	return []*regexp.Regexp{
		regexp.MustCompile(`\brm\s+-[rf]{1,2}\b`),
		regexp.MustCompile(`\bsudo\b`),
		regexp.MustCompile(`\bchmod\s+[0-7]{3,4}\b`),
		regexp.MustCompile(`\bchown\b`),
		regexp.MustCompile(`\b(shutdown|reboot|poweroff)\b`),
		regexp.MustCompile(`\bcurl\b.*\|\s*(sh|bash)`),
		regexp.MustCompile(`\bwget\b.*\|\s*(sh|bash)`),
		regexp.MustCompile(`\bssh\b.*@`),
		regexp.MustCompile(`\beval\b`),
	}
}

func (t *ShellTool) Name() string        { return "run_shell" }
func (t *ShellTool) Description() string { return "Run a shell command in the workspace directory. Use with care; dangerous commands are blocked." }
func (t *ShellTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{"type": "string", "description": "Shell command to run (e.g. ls -la, go build ./...)"},
		},
		"required": []string{"command"},
	}
}

func (t *ShellTool) Execute(ctx context.Context, args map[string]interface{}) *Result {
	raw, _ := args["command"].(string)
	command := strings.TrimSpace(raw)
	if command == "" {
		return ErrorResult("command is required", nil)
	}
	for _, re := range t.denyList {
		if re.MatchString(command) {
			return ErrorResult("command not allowed by policy", nil)
		}
	}
	dir := t.workspace
	if dir == "" {
		dir = "."
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return ErrorResult(fmt.Sprintf("workspace path: %v", err), err)
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = absDir
	if t.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.timeout)
		defer cancel()
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
		cmd.Dir = absDir
	}
	out, err := cmd.CombinedOutput()
	text := strings.TrimSuffix(string(out), "\n")
	if err != nil {
		return ErrorResult(fmt.Sprintf("command failed: %v\n%s", err, text), err)
	}
	return OkResult(text)
}
