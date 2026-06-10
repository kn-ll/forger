package monitoring

import "time"

// EventKind 表示监控事件类型。监控平台通过这些事件解释 run、工具、审批、
// artifact、上下文和成本。
type EventKind string

const (
	// EventRunStarted 表示 run 开始。
	EventRunStarted EventKind = "run.started"
	// EventRunCompleted 表示 run 结束。
	EventRunCompleted EventKind = "run.completed"
	// EventToolStarted 表示工具开始执行。
	EventToolStarted EventKind = "tool.started"
	// EventToolCompleted 表示工具结束执行。
	EventToolCompleted EventKind = "tool.completed"
	// EventApprovalCreated 表示产生审批请求。
	EventApprovalCreated EventKind = "approval.created"
	// EventArtifactCreated 表示产生 artifact。
	EventArtifactCreated EventKind = "artifact.created"
	// EventPolicyDenied 表示策略拒绝某个动作。
	EventPolicyDenied EventKind = "policy.denied"
	// EventContextAssembled 表示完成一次上下文装配。
	EventContextAssembled EventKind = "context.assembled"
	// EventCostObserved 表示记录一次成本或 token 统计。
	EventCostObserved EventKind = "cost.observed"
)

// Event 是监控平台的原始事件。它应该足够通用，能覆盖 CLI、TUI、后台 run 和子 Agent。
type Event struct {
	ID       string
	ThreadID string
	RunID    string
	Kind     EventKind
	Time     time.Time
	Severity string
	Message  string
	Fields   map[string]string
}

// RunSummary 是监控平台面向列表和概览页的聚合结果。
type RunSummary struct {
	ThreadID       string
	RunID          string
	Status         string
	ToolCalls      int
	Approvals      int
	Artifacts      int
	EstimatedCost  float64
	DurationMillis int64
	FailureReason  string
}
