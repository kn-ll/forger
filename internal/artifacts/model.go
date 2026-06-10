package artifacts

import "time"

// Kind 表示产物类型。Artifact 是 thread 中可复用的输出，不只是日志文本。
type Kind string

const (
	// KindDiff 表示代码 diff。
	KindDiff Kind = "diff"
	// KindVerification 表示测试、lint、build 等验证输出。
	KindVerification Kind = "verification"
	// KindReport 表示研究报告、审查报告或任务总结。
	KindReport Kind = "report"
	// KindDocument 表示文档类文件，例如 Markdown、docx、pptx、xlsx。
	KindDocument Kind = "document"
	// KindWebpage 表示抓取或整理后的网页内容。
	KindWebpage Kind = "webpage"
	// KindScreenshot 表示浏览器或应用截图。
	KindScreenshot Kind = "screenshot"
	// KindSubagentResult 表示子 Agent 的汇总输出。
	KindSubagentResult Kind = "subagent_result"
	// KindExternalRecord 表示 Jira、Confluence、GitLab 等外部系统对象。
	KindExternalRecord Kind = "external_record"
)

// Artifact 是可追踪、可复用的任务产物。后续 UI、监控和 Agent 上下文都应该通过
// Artifact 引用产物，而不是从日志里重新解析。
type Artifact struct {
	ID        string
	ThreadID  string
	RunID     string
	Kind      Kind
	Title     string
	URI       string
	CreatedAt time.Time
	Metadata  map[string]string
}
