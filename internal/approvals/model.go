package approvals

import "time"

// Status 表示审批请求状态。
type Status string

const (
	// StatusPending 表示等待用户或策略系统处理。
	StatusPending Status = "pending"
	// StatusApproved 表示审批通过。
	StatusApproved Status = "approved"
	// StatusRejected 表示审批拒绝。
	StatusRejected Status = "rejected"
	// StatusExpired 表示审批超时。
	StatusExpired Status = "expired"
)

// Request 表示一次风险动作审批。审批必须能关联回 thread、run 和 tool call。
type Request struct {
	ID         string
	ThreadID   string
	RunID      string
	ToolCallID string
	Reason     string
	Status     Status
	CreatedAt  time.Time
	ResolvedAt time.Time
}
