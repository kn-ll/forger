package tools

import "time"

// Risk 表示工具调用风险等级，用于审批、沙箱和监控分类。
type Risk string

const (
	// RiskRead 表示只读工具。
	RiskRead Risk = "read"
	// RiskWrite 表示会修改本地工作区或产物的工具。
	RiskWrite Risk = "write"
	// RiskExecute 表示会执行本地命令的工具。
	RiskExecute Risk = "execute"
	// RiskExternal 表示会访问或修改外部系统的工具。
	RiskExternal Risk = "external"
)

// Spec 描述一个可注册工具的公开契约。
type Spec struct {
	Name        string
	Description string
	Risk        Risk
}

// CallStatus 表示一次工具调用的运行状态。
type CallStatus string

const (
	// CallPending 表示工具调用已计划但尚未开始。
	CallPending CallStatus = "pending"
	// CallRunning 表示工具正在执行。
	CallRunning CallStatus = "running"
	// CallSucceeded 表示工具执行成功。
	CallSucceeded CallStatus = "succeeded"
	// CallFailed 表示工具执行失败。
	CallFailed CallStatus = "failed"
)

// Call 记录一次工具调用。它是监控、审批和 artifact 追踪的共同来源。
type Call struct {
	ID         string
	ThreadID   string
	RunID      string
	Tool       string
	Risk       Risk
	Status     CallStatus
	StartedAt  time.Time
	FinishedAt time.Time
	Input      map[string]any
	Output     string
	Error      string
}
