package agentbackend

import (
	"context"
	"fmt"
	"strings"
)

// PlaceholderRunner 用于 runtime 链路验证的占位 backend。
// TODO: 在接入真实执行框架后删除这个实现，改由 Eino adapter 等正式 backend 替代。
type PlaceholderRunner struct{}

// Run 生成占位 agent 回复，并在目标看起来像代码任务时试探性读取工作区树。
// TODO: 替换为真实 agent loop；当前实现只验证 backend -> toolruntime -> audit 链路。
func (PlaceholderRunner) Run(ctx context.Context, req Request, emitter Emitter) error {
	goal := strings.TrimSpace(req.Goal)
	lower := strings.ToLower(goal)
	if strings.Contains(lower, "code") ||
		strings.Contains(lower, "test") ||
		strings.Contains(lower, "bug") ||
		strings.Contains(lower, "fix") {
		// Phase 2 先用一次真实只读工具调用验证 runtime -> toolruntime -> thread audit 的整条链路。
		if _, err := emitter.CallTool(ctx, ToolRequest{
			Tool:  "file.tree",
			Input: map[string]any{"root": ".", "max_depth": 2},
		}); err != nil {
			return err
		}
	}
	return emitter.AppendMessage("agent", fmt.Sprintf("placeholder agent response: %s", goal))
}
