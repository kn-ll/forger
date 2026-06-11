package toolruntime

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/kn-ll/forger/internal/tools"
)

func RegisterBuiltins(registry *Registry, workspace string) {
	// 当前只注册只读工具，Phase 3 不允许静默引入任何写工具。
	registry.Register(fileTreeTool{workspace: workspace})
	registry.Register(fileReadTool{workspace: workspace})
	registry.Register(searchRGTool{workspace: workspace})
	registry.Register(gitDiffTool{workspace: workspace})
	registry.Register(gitLogTool{workspace: workspace})
}

// workspace 是工具运行时的项目根目录，同时承担默认上下文、路径边界和 git 执行上下文三种职责
type fileTreeTool struct{ workspace string }
type fileReadTool struct{ workspace string }
type searchRGTool struct{ workspace string }
type gitDiffTool struct{ workspace string }
type gitLogTool struct{ workspace string }

func (t fileTreeTool) Spec() tools.Spec {
	return tools.Spec{Name: "file.tree", Description: "List files under a workspace path", Risk: tools.RiskRead}
}
func (t fileReadTool) Spec() tools.Spec {
	return tools.Spec{Name: "file.read", Description: "Read a file from workspace", Risk: tools.RiskRead}
}
func (t searchRGTool) Spec() tools.Spec {
	return tools.Spec{Name: "search.rg", Description: "Search text with rg", Risk: tools.RiskRead}
}
func (t gitDiffTool) Spec() tools.Spec {
	return tools.Spec{Name: "git.diff", Description: "Read git diff", Risk: tools.RiskRead}
}
func (t gitLogTool) Spec() tools.Spec {
	return tools.Spec{Name: "git.log", Description: "Read git log", Risk: tools.RiskRead}
}

func (t fileTreeTool) Run(ctx context.Context, req CallRequest) (CallResult, error) {
	root, err := resolveWorkspacePath(t.workspace, stringInput(req.Input, "root", "."))
	if err != nil {
		return CallResult{}, err
	}
	maxDepth := intInput(req.Input, "max_depth", 2)
	var lines []string
	baseDepth := strings.Count(root, string(os.PathSeparator))
	err = filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		depth := strings.Count(path, string(os.PathSeparator)) - baseDepth
		if depth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(t.workspace, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		lines = append(lines, rel)
		return nil
	})
	if err != nil {
		return CallResult{}, err
	}
	sort.Strings(lines)
	return CallResult{Output: strings.Join(lines, "\n")}, nil
}

func (t fileReadTool) Run(ctx context.Context, req CallRequest) (CallResult, error) {
	path, err := resolveWorkspacePath(t.workspace, stringInput(req.Input, "path", ""))
	if err != nil {
		return CallResult{}, err
	}
	if path == t.workspace {
		return CallResult{}, fmt.Errorf("path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return CallResult{}, err
	}
	return CallResult{Output: string(data)}, nil
}

func (t searchRGTool) Run(ctx context.Context, req CallRequest) (CallResult, error) {
	pattern := stringInput(req.Input, "pattern", "")
	if strings.TrimSpace(pattern) == "" {
		return CallResult{}, fmt.Errorf("pattern is required")
	}
	cmd := exec.CommandContext(ctx, "rg", "--line-number", "--no-heading", pattern, ".")
	cmd.Dir = t.workspace
	out, err := cmd.CombinedOutput()
	if err != nil && len(out) == 0 {
		return CallResult{}, err
	}
	return CallResult{Output: string(out)}, nil
}

func (t gitDiffTool) Run(ctx context.Context, req CallRequest) (CallResult, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--", stringInput(req.Input, "path", "."))
	cmd.Dir = t.workspace
	out, err := cmd.CombinedOutput()
	if err != nil {
		return CallResult{}, err
	}
	return CallResult{Output: string(out)}, nil
}

func (t gitLogTool) Run(ctx context.Context, req CallRequest) (CallResult, error) {
	limit := strconv.Itoa(intInput(req.Input, "limit", 5))
	cmd := exec.CommandContext(ctx, "git", "log", "--oneline", "-n", limit)
	cmd.Dir = t.workspace
	out, err := cmd.CombinedOutput()
	if err != nil {
		return CallResult{}, err
	}
	return CallResult{Output: string(out)}, nil
}

func resolveWorkspacePath(workspace string, path string) (string, error) {
	if path == "" {
		return workspace, nil
	}
	joined := filepath.Clean(filepath.Join(workspace, path))
	workspaceClean := filepath.Clean(workspace)
	// 只读工具也必须受工作区边界约束，避免通过相对路径读取到 workspace 外内容。
	if joined != workspaceClean && !strings.HasPrefix(joined, workspaceClean+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes workspace: %s", path)
	}
	return joined, nil
}

func stringInput(input map[string]any, key string, fallback string) string {
	raw, ok := input[key]
	if !ok {
		return fallback
	}
	switch v := raw.(type) {
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

func intInput(input map[string]any, key string, fallback int) int {
	raw, ok := input[key]
	if !ok {
		return fallback
	}
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		parsed, err := strconv.Atoi(v)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func compactText(text string, limit int) string {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) <= limit {
		return trimmed
	}
	// artifact 先保存可直接展示的截断版本，避免 event stream 和默认展示层被大输出压垮。
	var buf bytes.Buffer
	buf.WriteString(trimmed[:limit])
	buf.WriteString("\n...[truncated]")
	return buf.String()
}
