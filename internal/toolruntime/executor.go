package toolruntime

import (
	"context"
	"time"

	"github.com/kn-ll/forger/internal/artifacts"
	"github.com/kn-ll/forger/internal/thread"
	"github.com/kn-ll/forger/internal/tools"
)

// Executor 统一执行工具，并把 ToolCall 与 artifact 写回 thread。
type Executor struct {
	registry  *Registry
	threads   thread.Store
	artifacts *artifacts.Manager
}

// NewExecutor 创建 tool executor。
func NewExecutor(registry *Registry, threads thread.Store, artifacts *artifacts.Manager) *Executor {
	return &Executor{registry: registry, threads: threads, artifacts: artifacts}
}

// Execute 执行工具。
func (e *Executor) Execute(ctx context.Context, threadID string, runID string, toolName string,
	input map[string]any) (tools.Call, []artifacts.Artifact, error) {
	tool, err := e.registry.Resolve(toolName)
	if err != nil {
		return tools.Call{}, nil, err
	}
	spec := tool.Spec()
	now := time.Now().UTC()
	call := tools.Call{
		ThreadID:  threadID,
		RunID:     runID,
		Tool:      spec.Name,
		Risk:      spec.Risk,
		Status:    tools.CallRunning,
		StartedAt: now,
		Input:     input,
	}
	call, err = e.threads.AppendToolCall(ctx, threadID, call)
	if err != nil {
		return tools.Call{}, nil, err
	}

	// ToolCall 先以 running 状态落盘，再执行工具，避免长执行或失败时丢失审计起点。
	result, runErr := tool.Run(ctx, CallRequest{ThreadID: threadID, RunID: runID, Input: input})
	call.Output = result.Output
	call.FinishedAt = time.Now().UTC()
	if runErr != nil {
		call.Status = tools.CallFailed
		call.Error = runErr.Error()
		updated, updateErr := e.threads.AppendToolCall(ctx, threadID, call)
		if updateErr != nil {
			return tools.Call{}, nil, updateErr
		}
		return updated, nil, runErr
	}
	call.Status = tools.CallSucceeded
	// 工具输出同步写成 artifact，后续 UI/上下文装配优先通过 artifact 复用，而不是重扫日志。
	// TODO: 为大输出保留完整 raw artifact，并把当前截断版和完整原文分层存储。
	art, err := e.artifacts.Create(ctx, artifacts.CreateInput{
		ThreadID: threadID,
		RunID:    runID,
		Kind:     artifacts.KindReport,
		Title:    spec.Name + " output",
		Content:  compactText(result.Output, 4000),
		Metadata: map[string]string{"tool": spec.Name},
	})
	if err != nil {
		return tools.Call{}, nil, err
	}
	art, err = e.threads.AppendArtifact(ctx, threadID, art)
	if err != nil {
		return tools.Call{}, nil, err
	}
	updated, err := e.threads.AppendToolCall(ctx, threadID, call)
	if err != nil {
		return tools.Call{}, nil, err
	}
	return updated, []artifacts.Artifact{art}, nil
}
