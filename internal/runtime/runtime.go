package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/kn-ll/forger/internal/agentbackend"
	"github.com/kn-ll/forger/internal/artifacts"
	"github.com/kn-ll/forger/internal/thread"
	"github.com/kn-ll/forger/internal/toolruntime"
	"github.com/kn-ll/forger/internal/tools"
)

// ThreadRuntime 是 Phase 2 的统一运行入口。
type ThreadRuntime struct {
	threads thread.Store
	backend agentbackend.Runner
	tools   *toolruntime.Executor
}

// StartRunRequest 定义启动一次 run 的输入。
type StartRunRequest struct {
	ThreadID string
	Goal     string
}

// New 创建运行时。
func New(threads thread.Store, backend agentbackend.Runner, tools *toolruntime.Executor) *ThreadRuntime {
	return &ThreadRuntime{threads: threads, backend: backend, tools: tools}
}

// StartRun 创建并执行一条 run。
func (r *ThreadRuntime) StartRun(ctx context.Context, req StartRunRequest) (thread.Thread, thread.Run, error) {
	threadID := strings.TrimSpace(req.ThreadID)
	if threadID == "" {
		return thread.Thread{}, thread.Run{}, fmt.Errorf("thread id is required")
	}
	item, err := r.threads.Get(ctx, threadID)
	if err != nil {
		return thread.Thread{}, thread.Run{}, err
	}
	run, err := r.threads.CreateRun(ctx, threadID, thread.CreateRunRequest{Goal: req.Goal})
	if err != nil {
		return thread.Thread{}, thread.Run{}, err
	}
	run, err = r.threads.UpdateRun(ctx, threadID, run.ID, thread.UpdateRunRequest{Status: thread.RunRunning})
	if err != nil {
		return thread.Thread{}, thread.Run{}, err
	}
	if err := r.backend.Run(ctx, agentbackend.Request{
		ThreadID: threadID,
		RunID:    run.ID,
		Goal:     req.Goal,
	}, &emitter{runtime: r, threadID: threadID, runID: run.ID}); err != nil {
		// backend 只能通过 emitter 回写 thread；一旦执行失败，由 runtime 统一收口 run 状态。
		run, _ = r.threads.UpdateRun(ctx, threadID, run.ID, thread.UpdateRunRequest{Status: thread.RunFailed})
		item, getErr := r.threads.Get(ctx, threadID)
		return item, run, getErr
	}
	// TODO: 接入真正的取消语义。当前只要 backend.Run 正常返回，就会直接写 succeeded；
	// 后续需要在这里检查 run 是否收到 cancel signal，避免 canceled run 被回写成 succeeded。
	run, err = r.threads.UpdateRun(ctx, threadID, run.ID, thread.UpdateRunRequest{Status: thread.RunSucceeded})
	if err != nil {
		return thread.Thread{}, thread.Run{}, err
	}
	item, err = r.threads.Get(ctx, threadID)
	return item, run, err
}

// CancelRun 当前只更新 run 状态。
// TODO: 维护 active run handle 和 context.CancelFunc，真正中断正在执行的 backend/tool。
func (r *ThreadRuntime) CancelRun(ctx context.Context, threadID string, runID string) (thread.Run, error) {
	return r.threads.UpdateRun(ctx, threadID, runID, thread.UpdateRunRequest{Status: thread.RunCanceled})
}

// ResumeRun 当前阶段直接报未实现。
// TODO: 为 interrupted/waiting_approval runs 设计 resume 语义，并恢复 backend 上下文。
func (r *ThreadRuntime) ResumeRun(context.Context, string, string) error {
	return fmt.Errorf("resume is not implemented yet")
}

type emitter struct {
	runtime  *ThreadRuntime
	threadID string
	runID    string
}

func (e *emitter) AppendMessage(role string, content string) error {
	// backend 不直接持久化 thread，只能通过 runtime 定义的 transcript 入口回写消息。
	_, err := e.runtime.threads.AppendMessage(context.Background(), e.threadID, thread.AppendMessageRequest{
		Role:    thread.MessageRole(role),
		Content: content,
		RunID:   e.runID,
	})
	return err
}

func (e *emitter) CallTool(ctx context.Context, request agentbackend.ToolRequest) (agentbackend.ToolResult, error) {
	if e.runtime.tools == nil {
		return agentbackend.ToolResult{}, fmt.Errorf("tool executor is not configured")
	}
	call, artifacts, err := e.runtime.tools.Execute(ctx, e.threadID, e.runID, request.Tool, request.Input)
	if err != nil {
		return agentbackend.ToolResult{}, err
	}
	var artifactIDs []string
	for _, artifact := range artifacts {
		artifactIDs = append(artifactIDs, artifact.ID)
	}
	// Tool 输出同时作为 transcript 消息暴露，保证 thread show/replay 能直接看到执行结果。
	_, _ = e.runtime.threads.AppendMessage(ctx, e.threadID, thread.AppendMessageRequest{
		Role:    thread.RoleTool,
		Content: call.Output,
		RunID:   e.runID,
	})
	return agentbackend.ToolResult{Output: call.Output, ArtifactIDs: artifactIDs}, nil
}

// Helper for tests.
func ToolNames(calls []tools.Call) []string {
	out := make([]string, 0, len(calls))
	for _, call := range calls {
		out = append(out, call.Tool)
	}
	return out
}

func ArtifactIDs(items []artifacts.Artifact) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.ID)
	}
	return out
}
