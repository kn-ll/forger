// Package storage 定义 Forger 的固定混合存储规则。
//
// Forger 不在 JSON 和 SQLite 之间二选一：文件层保存原始 transcript、JSONL
// 事件流和 artifact，SQLite 保存可查询索引、监控聚合和关系状态。这个分工是
// 产品规则，不是迁移阶段的临时安排。
package storage

// Layer 表示一种存储层。
type Layer string

const (
	// LayerConfig 保存 TOML 配置。
	LayerConfig Layer = "config"
	// LayerJSON 保存轻量 JSON 状态和缓存。
	LayerJSON Layer = "json"
	// LayerJSONL 保存 append-only transcript、事件流和审计日志。
	LayerJSONL Layer = "jsonl"
	// LayerSQLite 保存可查询产品状态和监控索引。
	LayerSQLite Layer = "sqlite"
	// LayerFiles 保存附件、截图、生成文件、skill 包和 artifact。
	LayerFiles Layer = "files"
)

// Layout 描述 Forger home 下的标准存储路径。
//
// 规则固定如下：
//  1. `config.toml` 保存用户和项目配置，例如模型默认值、审批策略、沙箱模式、
//     MCP 配置和功能开关。它不保存运行期业务状态。
//  2. `auth.json` 保存文件型凭据或凭据元数据，例如 token、刷新信息，或指向系统
//     凭据存储的引用。它只解决认证问题，不承载 thread、run 或 memory 状态。
//  3. `history.jsonl` 保存 CLI 输入历史，用于历史回填、输入建议和命令回放。它不是
//     thread transcript，删除后不应该影响 thread 的业务状态恢复。
//  4. `session_index.jsonl` 保存 thread 的轻量索引和归档索引，例如 `thread_id`、
//     `title`、`status`、`created_at`、`updated_at`。它用于快速列出 thread，
//     而不是保存完整对话。
//  5. `next_id.json` 保存文件层的下一个 thread 编号，例如下一个可分配的
//     `thr-000123`。它只负责稳定分配 ID，不保存 thread 内容。
//  6. `sessions/` 保存按 thread ID 分片的原始 transcript 和事件流 JSONL 文件，
//     例如 `sessions/<thread-id>.jsonl`。这里是 thread 的文件层真相来源，逐行追加
//     用户消息、Agent 回复、工具结果、审批事件和运行事件。
//  7. `state.sqlite` 保存 thread/run/tool/approval/artifact 的核心关系状态，用于
//     产品查询，例如列出最近 thread、查某次 run 的工具调用、查 artifact 归属。
//  8. `logs.sqlite` 保存监控事件、审计日志、错误分类、重试标记和成本统计索引。
//     它面向监控平台和告警查询，不保存完整 transcript 原文。
//  9. `memories.sqlite` 保存长期 memory、memory 检索索引和 memory 元数据，例如
//     scope、tags、evidence、confidence、expires_at 和 conflict_key。
//  10. `artifacts/` 保存 Forger 运行后生成的输出文件，例如 diff、报告、截图、
//     表格、文档和网页快照。SQLite 中只保存它们的元数据和引用关系。
//  11. `attachments/` 保存用户或外部系统提供给 Forger 的输入附件，例如 PDF、
//     图片、日志文件和下载材料。它和 `artifacts/` 的区别是：前者是输入，后者是输出。
type Layout struct {
	Home             string
	ConfigPath       string
	AuthPath         string
	HistoryPath      string
	SessionIndexPath string
	NextIDPath       string
	StateDBPath      string
	LogsDBPath       string
	MemoriesDBPath   string
	SessionsDir      string
	ArtifactsDir     string
	AttachmentsDir   string
}

// DefaultLayout 返回固定的 Forger 存储布局。调用方传入 home，通常是
// ~/.forger 或通过环境变量覆盖后的 Forger home。
func DefaultLayout(home string) Layout {
	return Layout{
		Home:             home,
		ConfigPath:       home + "/config.toml",
		AuthPath:         home + "/auth.json",
		HistoryPath:      home + "/history.jsonl",
		SessionIndexPath: home + "/session_index.jsonl",
		NextIDPath:       home + "/next_id.json",
		StateDBPath:      home + "/state.sqlite",
		LogsDBPath:       home + "/logs.sqlite",
		MemoriesDBPath:   home + "/memories.sqlite",
		SessionsDir:      home + "/sessions",
		ArtifactsDir:     home + "/artifacts",
		AttachmentsDir:   home + "/attachments",
	}
}
