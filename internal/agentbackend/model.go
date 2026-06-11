package agentbackend

import "context"

// ToolRequest 描述 backend 计划发起的一次工具调用。
type ToolRequest struct {
	Tool  string
	Input map[string]any
}

// ToolResult 是一次工具调用返回给 backend 的结果。
type ToolResult struct {
	Output       string
	ArtifactIDs  []string
	ErrorMessage string
}

// Runner 是外部执行框架的统一适配边界。
type Runner interface {
	Run(context.Context, Request, Emitter) error
}

// Request 是 backend 的运行输入。
type Request struct {
	ThreadID string
	RunID    string
	Goal     string
}

// Emitter 是 backend 与 Forger runtime 交互的最小接口。
type Emitter interface {
	AppendMessage(role string, content string) error
	CallTool(ctx context.Context, request ToolRequest) (ToolResult, error)
}
