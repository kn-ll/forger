package thread

import (
	"strings"
	"time"

	"github.com/kn-ll/forger/internal/artifacts"
	"github.com/kn-ll/forger/internal/tools"
)

// Status 表示线程生命周期状态。第一阶段只区分打开和归档，后续可以扩展为
// pinned、blocked 等产品状态，但不应该和单次 run 状态混用。
type Status string

const (
	// StatusOpen 表示线程仍可继续对话和追加 run。
	StatusOpen Status = "open"
	// StatusArchived 表示线程已从主要工作列表移除，但历史仍可审计。
	StatusArchived Status = "archived"
)

// MessageRole 表示一条消息在 thread 中的来源。
type MessageRole string

const (
	// RoleUser 是用户输入。
	RoleUser MessageRole = "user"
	// RoleAgent 是主 Agent 的回复。
	RoleAgent MessageRole = "agent"
	// RoleSystem 是系统或运行时写入的控制消息。
	RoleSystem MessageRole = "system"
	// RoleTool 是工具调用结果。
	RoleTool MessageRole = "tool"
	// RoleApproval 是审批请求或审批结果。
	RoleApproval MessageRole = "approval"
	// RoleMonitor 是监控平台写入的状态摘要。
	RoleMonitor MessageRole = "monitor"
)

// Thread 是 Forger 的第一产品对象。它承载长期任务上下文，并把消息、run、
// 工具调用、审批和产物连接到同一个可恢复会话里。
type Thread struct {
	ID        string
	Title     string
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
	Messages  []Message // 这个 thread 里交流了什么
	Runs      []Run     // 这个 thread 里执行了哪些任务，它们现在什么状态
	ToolCalls []tools.Call
	Artifacts []artifacts.Artifact
}

// Message 是 thread 内的最小上下文单元。后续可以增加 artifact refs、tool refs
// 和 token 统计，但第一阶段先保持简单。
type Message struct {
	ID        string
	Role      MessageRole
	Content   string
	RunID     string
	CreatedAt time.Time
}

// RunStatus 表示一次 Agent 执行的状态。它只描述单次运行，不描述 thread 生命周期。
type RunStatus string

const (
	// RunPending 表示 run 已创建但尚未开始。
	RunPending RunStatus = "pending"
	// RunRunning 表示 run 正在执行。
	RunRunning RunStatus = "running"
	// RunSucceeded 表示 run 正常完成。
	RunSucceeded RunStatus = "succeeded"
	// RunFailed 表示 run 执行失败。
	RunFailed RunStatus = "failed"
	// RunCanceled 表示 run 被用户或系统取消。
	RunCanceled RunStatus = "canceled"
)

// Run 表示 thread 内的一次 Agent 执行。未来 tool call、approval 和 artifact 都会
// 通过 RunID 关联到这次执行。
type Run struct {
	ID        string
	ThreadID  string
	Goal      string
	Status    RunStatus
	StartedAt time.Time
	EndedAt   time.Time
}

// CreateRequest 是创建 thread 的输入契约。
type CreateRequest struct {
	Title string
}

// AppendMessageRequest 是向 thread transcript 追加消息的输入。
type AppendMessageRequest struct {
	Role    MessageRole
	Content string
	RunID   string
}

// Validate 校验消息追加请求。
func (r AppendMessageRequest) Validate() error {
	if strings.TrimSpace(string(r.Role)) == "" {
		return ErrMessageRoleRequired
	}
	if strings.TrimSpace(r.Content) == "" {
		return ErrMessageContentRequired
	}
	return nil
}

// CreateRunRequest 是在 thread 中创建 run 的输入。
type CreateRunRequest struct {
	Goal string
}

// Validate 校验 run 创建请求。
func (r CreateRunRequest) Validate() error {
	if strings.TrimSpace(r.Goal) == "" {
		return ErrRunGoalRequired
	}
	return nil
}

// UpdateRunRequest 是更新 run 状态的输入。
type UpdateRunRequest struct {
	Status RunStatus
}

// Validate 校验 run 状态更新请求。
func (r UpdateRunRequest) Validate() error {
	if strings.TrimSpace(string(r.Status)) == "" {
		return ErrRunStatusRequired
	}
	return nil
}

// Validate 校验创建请求，保证 thread 至少有可展示标题。
func (r CreateRequest) Validate() error {
	if strings.TrimSpace(r.Title) == "" {
		return ErrTitleRequired
	}
	return nil
}
