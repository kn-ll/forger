# Forger 线程化 Agent 工作台重新设计 Plan

## 0. 文档目的

本文不是对既有提示的整理，而是基于当前 Forger 代码状态，对产品形态、架构边界、核心数据模型、模块拆分、执行链路和阶段路线进行重新设计。

目标是把 Forger 做成一个本地优先的线程化 agent 工作台，能力形态对齐 Codex / Claude Code：用户面对的是持续工作的 agent workspace，而不是一组割裂的 CLI 命令。

当前仓库已经按新产品方向重建，已有 `thread`、`tools`、`approvals`、`artifacts`、`monitoring`、`storage` 等基础模型，并且从第一天起就采用正式产品结构，而不是临时原型结构。

## 1. 产品定位

Forger 的产品定位：

```text
Forger = Thread Runtime + Tool Runtime + Skills + Context + Memory + Artifacts + Audit
```

核心判断：

- Coding 是 Forger 的第一高价值工作流，但不是唯一工作流。
- `run`、`code`、`research`、`docs`、`browser`、`external system` 都应该是 thread 中的一类 intent。
- 用户不应该理解一堆内部命令；用户应该理解 thread、run、tool call、approval、artifact、skill。
- RAG 不应该主导代码理解；代码理解优先依赖确定性搜索、文件树、符号、AST / LSP、git 历史。
- Memory 不是聊天记录归档；Memory 是经过治理的长期偏好、规则、事实和决策。
- 安全、审批、审计、恢复能力是产品底座，不是后补功能。

## 2. 设计原则

1. Thread first：所有用户任务都必须能归属到 thread。
2. Tool first：agent 不能隐藏执行副作用，所有真实操作都必须通过 tool runtime。
3. Audit first：每次 run、tool call、approval、artifact 都必须可追踪。
4. Local first：原始 transcript 和 artifacts 保存在文件系统，SQLite 做查询索引。
5. Deterministic before semantic：代码场景先用确定性工具，再考虑 RAG。
6. Skills as contracts：skill 是工作流契约，不是散落 prompt。
7. Memory is governed：默认不把对话自动变成长期记忆。
8. Small product surface：主入口少，复杂维护能力放到 debug/admin。
9. Frameworks behind runtime boundaries：外部 agent 框架只能作为执行后端或编排适配器，不能定义 Forger 的核心产品模型。

## 3. 当前实现评估

### 3.1 已具备的基础

| 模块 | 当前状态 | 设计评价 |
|---|---|---|
| `cmd/forger` | 支持 `thread new`、`thread list` | 入口足够薄，适合继续让 CLI 只做 runtime 调度 |
| `internal/thread` | 有 Thread、Message、Run、FileStore | 方向正确，但缺 event append / replay / run lifecycle |
| `internal/tools` | 有 Spec、Risk、Call | 模型可保留，需要补 registry、schema、executor |
| `internal/approvals` | 有 Request 模型 | 需要补 policy、decision、持久化和 CLI / UI 交互 |
| `internal/artifacts` | 有 Artifact 模型 | 需要补 artifact manager、hash、preview、producer |
| `internal/monitoring` | 有 Event、RunSummary | 需要补 writer、query、audit 和 failure 分类 |
| `internal/storage` | 已固定目录布局与文件命名 | 方向正确，需要继续补齐 JSONL event envelope、SQLite schema 和索引写入器 |

### 3.2 当前关键缺口

- 没有统一 `ThreadRuntime`，所以还不能把 run、tool、approval、artifact 串成闭环。
- `sessions/<thread-id>.jsonl` 已切到正式文件层，但事件 envelope、run 事件和 tool 事件还没有补齐。
- 没有 tool registry，无法统一工具 schema、risk、approval、trace、artifact。
- 没有 deterministic code search，coding workflow 还缺主检索路径。
- 没有 context assembler，无法稳定解释“这次模型看到了什么”。
- 没有 skill runtime，工作流能力无法产品化。
- 没有 memory governance，长期上下文边界不清。
- `state.sqlite`、`logs.sqlite`、`memories.sqlite` 责任已经固定，但还没有真正写入查询索引。

## 4. 目标产品面

### 4.1 主 UX

主 UX 围绕 workspace 和 thread：

```text
forger                         # 打开默认工作台，或进入交互式 thread workspace
forger thread new "<title>"    # 创建 thread
forger run "<goal>"            # 在当前或新 thread 中启动 run
forger thread list
forger thread show <thread-id>
forger thread open <thread-id>
forger approvals
forger artifacts
forger skills
forger memory
forger tools
forger settings
```

兼容 UX：

```text
forger code "<goal>"            # 等价于 forger run --skill coding.implement
forger review                   # 等价于 forger run --skill coding.review
forger explain <file-or-symbol> # 等价于 forger run --skill coding.explain
```

内部 / 高级 UX：

```text
forger debug storage
forger debug replay <thread-id>
forger debug tools
forger debug knowledge
forger debug index
```

### 4.2 不作为主产品面的内容

以下能力不进入主产品面：

- gateway。
- scheduler / tasks。
- 大型 admin 命令树。
- 复杂 RAG eval。
- embedding preparation。
- 单独的 codingrun 产品入口。

## 5. 核心对象重新设计

### 5.1 Thread

Thread 是长期任务容器。

必须支持：

- open / archived / pinned / blocked 状态。
- title、summary、tags。
- 最近 active run。
- artifacts、memories、subagent tasks 的引用。
- 从 JSONL event replay。

### 5.2 Message

Message 是 transcript 单元，不应该承担 tool call 的全部结构化字段。

角色：

- `user`
- `agent`
- `system`
- `tool`
- `approval`
- `monitor`

Message 可以引用：

- `run_id`
- `tool_call_id`
- `artifact_ids`
- `approval_request_id`

### 5.3 Run

Run 是 thread 内的一次 agent 执行。

状态：

- `pending`
- `assembling_context`
- `running`
- `waiting_approval`
- `succeeded`
- `failed`
- `canceled`

Run 必须记录：

- goal。
- selected skill。
- model/provider。
- context pack ID。
- tool call 数量。
- artifacts。
- failure reason。
- token/cost 统计。

### 5.4 ToolCall

ToolCall 是可审计执行单元。

必须记录：

- tool name 和 version。
- input schema version。
- redacted input。
- risk。
- approval decision。
- sandbox policy snapshot。
- started / finished 时间。
- output summary。
- raw output artifact。
- error category。

### 5.5 ApprovalRequest

ApprovalRequest 是用户或策略系统对风险动作的决策。

必须支持：

- approve once。
- approve for thread。
- approve for workspace policy。
- reject。
- expire。
- attach reason。

### 5.6 Artifact

Artifact 是 thread 可复用输出。

必须记录：

- stable ID。
- kind。
- title。
- content URI。
- content hash。
- producer run / tool / subagent。
- metadata。
- preview text。
- created_at。

### 5.7 Skill

Skill 是声明式工作流契约。

Skill 不是 prompt 文件。它必须声明：

- 适用场景。
- 输入类型。
- 输出类型。
- 允许工具。
- 风险等级。
- 依赖资源。
- 验证方法。
- 示例任务。

### 5.8 Memory

Memory 是长期上下文资产。

必须包含：

- scope。
- kind。
- content。
- evidence。
- confidence。
- conflict key。
- expiry。
- review status。

默认只使用 promoted memory。

### 5.9 ContextPack

ContextPack 是一次 run 的上下文快照。

必须记录：

- sources。
- inclusion reason。
- token estimate。
- freshness。
- priority。
- truncation decision。

这保证 run 可复盘：模型为什么看到这些信息，而不是别的信息。

## 6. 运行链路设计

### 6.1 一次普通 run 的流程

```text
User Intent
  |
  v
Resolve or Create Thread
  |
  v
Create Run
  |
  v
Select Skill
  |
  v
Assemble ContextPack
  |
  v
Agent Loop
  |
  +--> Tool Runtime
  |       |
  |       +--> Risk / Sandbox / Approval
  |       +--> Execute Tool
  |       +--> Emit ToolCall Events
  |       +--> Create Artifacts
  |
  v
Write Agent Message
  |
  v
Finalize Run
```

### 6.2 tool call 流程

```text
Tool Request
  |
  v
Registry Resolve
  |
  v
Input Validate
  |
  v
Risk Classify
  |
  v
Policy Check
  |
  +-- denied --> Audit + Return Error
  |
  +-- approval required --> Create ApprovalRequest --> Wait
  |
  v
Sandbox Prepare
  |
  v
Execute
  |
  v
Capture Output
  |
  v
Create Artifact if needed
  |
  v
Emit Monitoring / Audit
```

### 6.3 context assembly 流程

```text
Run Goal
  |
  v
Collect Explicit Inputs
  |
  v
Collect Thread Recent Messages
  |
  v
Collect Active Artifacts
  |
  v
Run Deterministic Code Search if coding intent
  |
  v
Load Skill Instructions
  |
  v
Load Promoted Memory
  |
  v
Retrieve Knowledge Docs if needed
  |
  v
Rank / Budget / Trace
  |
  v
Persist ContextPack
```

## 7. 模块拆分

### 7.1 当前模块演进

| 模块 | 调整方向 |
|---|---|
| `internal/thread` | 增加事件模型、append API、replay、thread repository |
| `internal/tools` | 保留模型，增加 schema、capability、tool identity |
| `internal/approvals` | 增加 policy engine、decision store、approval scope |
| `internal/artifacts` | 增加 manager、content store、hash、preview |
| `internal/monitoring` | 增加 event writer、query、audit、failure classification |
| `internal/storage` | 增加正式 SQLite schema、索引写入器、rebuild index |
| `cmd/forger` | 继续保持薄入口，只做参数解析和 runtime 调用 |

### 7.2 新增模块

| 模块 | 职责 |
|---|---|
| `internal/runtime` | ThreadRuntime、RunRuntime、agent loop 边界 |
| `internal/agentbackend` | 外部执行框架适配边界，负责把 Eino 等执行后端映射回 Forger runtime event |
| `internal/events` | append-only event 类型、编码、replay |
| `internal/toolruntime` | registry、executor、policy integration、trace |
| `internal/sandbox` | path、shell、network、env 安全策略 |
| `internal/contextpack` | 上下文装配、预算、trace |
| `internal/codesearch` | 文件树、rg、git、symbol、AST / LSP 接口 |
| `internal/skills` | skill manifest、loader、matcher、validator |
| `internal/memory` | memory store、candidate、promotion、conflict |
| `internal/knowledge` | 文档 RAG backend |
| `internal/mcp` | MCP server config、tool adapter、mutation policy |
| `internal/subagents` | subagent task、parallel execution、worktree isolation |
| `internal/config` | config.toml、project policy、feature flags |
| `internal/providers` | model provider abstraction |
| `internal/tui` | thread workspace UI |

### 7.3 外部框架接入策略

Forger 主实现语言为 Go，因此执行框架优先级如下：

1. 首选接入 Eino。
2. 不把 Google ADK 作为近阶段主 runtime 依赖。
3. 不允许外部框架接管 thread/run/message/approval/artifact/event schema。

接入边界：

- Forger 自己维护 `Thread`、`Message`、`Run`、`ToolCall`、`ApprovalRequest`、`Artifact`。
- Forger 自己维护 `sessions/<thread-id>.jsonl`、SQLite 查询层和审计层。
- Eino 仅负责 agent execution、workflow orchestration、multi-step loop、subagent 编排等执行能力。
- Eino 的执行结果必须映射回 Forger session events、tool calls、approvals 和 artifacts。

结论：

- Eino 是执行层适配器，不是产品内核。
- Forger 不使用框架原生 session/state 模型替代自身线程模型。

## 8. 存储设计

采用固定混合存储：

```text
~/.forger/
  config.toml
  auth.json
  history.jsonl
  session_index.jsonl
  next_id.json
  state.sqlite
  logs.sqlite
  memories.sqlite
  sessions/
    thr-000001.jsonl
  artifacts/
    thr-000001/
  attachments/
```

### 8.1 JSONL 是原始事实层

`session_index.jsonl` 只负责 thread 轻量索引：

- thread ID。
- title。
- status。
- created_at。
- updated_at。

`next_id.json` 只负责分配下一个 thread 编号，保证命名稳定且可跨进程恢复。

`sessions/<thread-id>.jsonl` 保存 append-only events：

- `thread.created`
- `thread.updated`
- `message.appended`
- `run.created`
- `run.status_changed`
- `contextpack.created`
- `toolcall.created`
- `toolcall.updated`
- `approval.created`
- `approval.resolved`
- `artifact.created`
- `subagent.created`
- `subagent.completed`

设计要求：

- JSONL 可以独立 replay 出 thread 状态。
- SQLite 损坏时可以由 JSONL 重建索引。
- event 必须有 version，便于迁移。
- `session_index.jsonl` 不是完整事实源，完整事实源只在 `sessions/<thread-id>.jsonl`。

### 8.2 SQLite 是查询层

`state.sqlite`：

- `threads`
- `messages`
- `runs`
- `context_packs`
- `tool_calls`
- `approval_requests`
- `artifacts`
- `skills`
- `subagent_tasks`

`logs.sqlite`：

- `monitoring_events`
- `audit_events`
- `failure_events`
- `cost_events`
- `tool_latency_daily`

`memories.sqlite`：

- `memories`
- `memory_evidence`
- `memory_conflicts`
- `memory_reviews`

### 8.3 Artifact 文件层

artifact 文件路径建议：

```text
artifacts/<thread-id>/<artifact-id>/
  content
  metadata.json
  preview.txt
```

规则：

- SQLite 只保存 metadata 和 URI。
- 大文本、截图、文档、diff、日志都放文件系统。
- content hash 用于去重和审计。

## 9. Tool Runtime 设计

### 9.1 Tool 接口

建议接口：

```go
type Tool interface {
    Spec() Spec
    Validate(ctx context.Context, input map[string]any) error
    Run(ctx context.Context, req RunRequest) (RunResult, error)
}
```

`Spec` 至少包含：

- name。
- version。
- description。
- input schema。
- output schema。
- risk。
- side effects。
- required capabilities。
- timeout default。

### 9.2 内置工具优先级

P0 read-only：

- `file.tree`
- `file.read`
- `search.rg`
- `git.diff`
- `git.log`
- `git.show`

P0 controlled mutation：

- `file.patch`
- `artifact.write`

P0 execute：

- `shell.run`，默认需要 approval。
- `test.run`，可作为受限 shell wrapper。

P1：

- `mcp.call`
- `web.fetch`
- `browser.open`
- `browser.click`
- `docs.create`
- `sheets.create`
- `slides.create`

### 9.3 风险等级

| Risk | 含义 | 默认策略 |
|---|---|---|
| `read` | 只读本地或远端信息 | 允许，记录 audit |
| `write` | 修改 workspace 或 artifact | workspace 内可按策略允许，其他需要审批 |
| `execute` | 执行本地命令 | 默认审批 |
| `external` | 访问外部系统 | read 可配置，mutation 默认审批 |
| `secret` | 访问凭据或敏感信息 | 默认审批并脱敏 |
| `destructive` | 删除、覆盖、重置、发布 | 强制审批 |

## 10. Approval / Sandbox / Audit 设计

### 10.1 Approval policy

Policy 输入：

- tool spec。
- risk。
- input。
- workspace root。
- current thread。
- selected skill。
- user settings。
- prior approval scope。

Policy 输出：

- allow。
- deny。
- require approval。

Approval scope：

- once。
- run。
- thread。
- workspace。

### 10.2 Sandbox policy

最小策略：

- 文件读写限制在 workspace roots 和 `.forger` artifact 目录。
- 禁止默认读取明显 secret 文件，除非 approval。
- shell 默认无交互、有限 timeout。
- shell 输出脱敏。
- 网络能力可按工具分类控制。

### 10.3 Audit event

每个风险动作写 audit event：

- actor。
- thread_id。
- run_id。
- tool_call_id。
- action。
- risk。
- decision。
- redacted_input。
- output_summary。
- artifacts。
- timestamp。

## 11. Deterministic Code Search 设计

`internal/codesearch` 是 coding workflow 的主上下文来源。

第一版能力：

- 文件树扫描，尊重 `.gitignore`。
- `rg` 搜索。
- git diff / log / show。
- Go package / test 文件发现。
- 简单 Go symbol 提取。

查询接口：

```go
type SearchRequest struct {
    Root string
    Query string
    Intent string
    Paths []string
    IncludeGit bool
}
```

输出：

- matched files。
- line ranges。
- symbol hints。
- git evidence。
- ranking reason。

设计边界：

- 不依赖 embedding 才能工作。
- 不把 RAG 作为代码主召回。
- 搜索结果必须能解释和复现。

## 12. ContextPack 设计

ContextPack 是 run 的输入快照。

Context sources：

1. thread recent messages。
2. explicit files / attachments / URLs。
3. active artifacts。
4. code search results。
5. skill instructions。
6. promoted memory。
7. knowledge retrieval。
8. older summary。

每个 source 记录：

- type。
- URI 或 ID。
- title。
- content summary。
- token estimate。
- inclusion reason。
- priority。
- truncation state。

ContextPack 必须持久化，方便 run replay 和 debug。

## 13. Skill 设计

### 13.1 Skill manifest

建议格式：

```toml
id = "coding.implement"
name = "Coding Implement"
version = "0.1.0"
description = "Implement code changes in a repository."
risk = "write"
inputs = ["thread", "files", "goal"]
outputs = ["diff", "verification", "summary"]
allowed_tools = [
  "file.tree",
  "file.read",
  "search.rg",
  "git.diff",
  "file.patch",
  "test.run"
]
validation = [
  "diff_created_or_explained",
  "verification_attempted"
]
```

### 13.2 内置 skill

P0：

- `coding.implement`
- `coding.review`
- `coding.explain`
- `project.onboard`

P1：

- `research.web`
- `docs.create`
- `browser.test`
- `memory.review`

### 13.3 Skill runtime

职责：

- intent matching。
- instruction loading。
- allowed tool set 约束。
- expected artifact 声明。
- validation 执行。
- eval case 管理。

## 14. Memory 设计

Memory 生命周期：

```text
candidate -> promoted -> expired
          -> rejected
```

默认规则：

- agent 可以提出 memory candidate。
- 用户确认或安全规则 promotion 后才进入默认上下文。
- memory 必须有 evidence。
- conflict key 相同的 memory 必须做冲突处理。

Memory scope：

- global。
- workspace。
- repo。
- thread。
- project。

Memory kind：

- preference。
- rule。
- decision。
- fact。
- external_reference。

Memory 不保存完整聊天历史，thread transcript 已经承担历史记录职责。

## 15. RAG / Knowledge 设计

RAG 是文档知识后端，不是 Forger 的主产品中心。

适用：

- PRD。
- TD。
- Confluence / Wiki。
- PDF。
- 长日志。
- 会议纪要。

不适用：

- 代码主检索。
- 替代 git diff。
- 替代文件读取。

Knowledge result 必须带 evidence：

- source URI。
- chunk ID。
- score。
- quote span。
- freshness。

## 16. Subagents 设计

Subagent 是 thread runtime 下的 delegated run。

类型：

- planner。
- researcher。
- reviewer。
- worker。

执行要求：

- 归属 parent thread / run。
- 有独立 context pack。
- 有独立 tool policy。
- 输出 artifact。
- parent run 可引用其 artifact。

Worker subagent 如果修改代码，必须使用隔离 worktree，并把 diff 作为 artifact 返回。

## 17. UI 工作台设计

### 17.1 TUI 优先

第一版可以做 TUI，因为当前产品是本地 CLI。

核心视图：

- Thread list。
- Active thread transcript。
- Run timeline。
- Tool calls。
- Pending approvals。
- Artifacts。
- Context sources。
- Memory review。

### 17.2 Web / Desktop

如果做 Web / Desktop，仍然复用同一套 runtime 和 store，不另起产品模型。

## 18. 分阶段实施计划

### Phase 1：补齐正式 Thread 文件层

目标：让 thread 文件层完整符合正式存储规则，并可稳定恢复。

状态：已完成。

任务：

1. 增加 `internal/events`。
2. 定义 session JSONL event envelope。
3. 扩展 `thread.Store`：append message、create run、update run。
4. 实现 event replay。
5. 增加 `thread show` CLI。
6. 补齐 `session_index.jsonl`、`next_id.json`、`sessions/<thread-id>.jsonl` 一致性测试。

验收：

- JSONL 能 replay 出完整 thread。
- thread 支持追加消息和 run。
- 现有 `thread new/list` 继续工作。

实现文档：

- 详见 [AGENT_WORKBENCH_PHASE1.md](./AGENT_WORKBENCH_PHASE1.md)

### Phase 2：RunRuntime 骨架

目标：建立统一运行入口，但暂不接真实模型。

任务：

1. 新增 `internal/runtime`。
2. 新增 `internal/agentbackend` 抽象，但先只提供占位 backend。
3. 实现 `StartRun`、`CancelRun`、`ResumeRun` 接口。
4. run 状态写入 event stream。
5. monitoring 写入 run started / completed。
6. CLI 增加 `forger run "<goal>"`，先生成占位 agent response。

验收：

- `forger run` 会创建 thread 或复用 thread。
- run lifecycle 可在 thread 中查看。
- run event 可审计。

### Phase 3：Tool Runtime 和只读工具

目标：统一工具注册、执行、审计。

任务：

1. 新增 `internal/toolruntime`。
2. 实现 registry。
3. 实现 `file.tree`、`file.read`、`search.rg`、`git.diff`、`git.log`。
4. ToolCall 写入 event stream。
5. Tool output 可写 artifact。

验收：

- 每个工具调用都有 ToolCall record。
- 只读 coding context 可通过工具获得。
- 工具失败有结构化错误。

### Phase 4：Approval / Sandbox / Artifact

目标：打通风险动作安全闭环。

任务：

1. 新增 `internal/sandbox`。
2. 扩展 `internal/approvals` policy。
3. 实现 `file.patch` 和 `shell.run`。
4. shell 和 write 默认走 approval。
5. 扩展 artifact manager。
6. CLI 增加 approval respond 和 artifact list/show。

验收：

- 风险动作不会静默执行。
- approval 决策持久化。
- diff / verification log 成为 artifact。

### Phase 5：Code Search 和 ContextPack

目标：形成 coding workflow 的确定性上下文路径。

任务：

1. 新增 `internal/codesearch`。
2. 新增 `internal/contextpack`。
3. code intent 自动触发 code search。
4. context pack 持久化。
5. context source 可在 CLI 查看。

验收：

- coding run 能解释上下文来源。
- RAG 不参与默认代码检索。
- 搜索结果有文件、行号、ranking reason。

### Phase 6：Skill Runtime

目标：把工作流能力产品化。

任务：

1. 新增 `internal/skills`。
2. 定义 skill manifest。
3. 加入内置 `coding.implement`、`coding.review`、`coding.explain`。
4. run 支持 `--skill`。
5. skill 限制 allowed tools。
6. skill validation 生成 artifact。

验收：

- `forger code` 成为 `forger run --skill coding.implement` 的别名。
- skill 可以控制工具权限。
- run 输出符合 skill expected artifacts。

### Phase 7：Model Provider 和最小 Agent Loop

目标：接入真实 LLM，使 Forger 可以执行端到端 coding 任务。

任务：

1. 新增 `internal/providers`。
2. 支持至少一个 provider。
3. 新增 `internal/agentbackend/eino`，作为首个真实执行 backend。
4. Agent loop 支持 tool call。
5. 支持 patch proposal / apply。
6. 支持验证命令。

验收：

- agent 可以读取代码、提出修改、申请审批、应用 patch、运行验证。
- 所有步骤都有 trace、tool call、artifact。

### Phase 8：Memory / Knowledge / Subagents

目标：补齐长期连续性、文档知识和多 agent 协作。

任务：

1. 新增 memory candidate / promotion。
2. 新增 knowledge backend。
3. 新增 subagent task。
4. 支持 researcher / reviewer 并行。
5. subagent 输出 artifact。

验收：

- memory 默认不自动污染上下文。
- 文档 RAG 可作为 context source。
- subagent 结果可复用和审计。

### Phase 9：TUI Thread Workspace

目标：让产品体验从命令集合转向工作台。

任务：

1. thread 列表视图。
2. active thread transcript。
3. run timeline。
4. tool call 面板。
5. approval 面板。
6. artifact 面板。
7. context source 面板。

验收：

- 用户能主要通过 thread workspace 完成任务。
- 命令行仍可作为自动化入口。

## 19. 近期可执行 Backlog

最建议立刻做的 12 个任务：

1. 定义 event envelope。
2. 把 `sessions/<thread-id>.jsonl` 改成 append-only event。
3. 为 `thread.Store` 增加 `AppendMessage`。
4. 为 `thread.Store` 增加 `CreateRun` 和 `UpdateRunStatus`。
5. 实现 event replay。
6. 增加 `forger thread show`。
7. 新增 `internal/runtime` 并实现占位 run。
8. 新增 `internal/toolruntime.Registry`。
9. 实现 `file.tree` 和 `file.read`。
10. 实现 `search.rg`。
11. ToolCall 事件化。
12. Artifact manager 最小实现。

这组任务完成后，Forger 就会从“只有基础 thread 能力”变成“能承载真实 agent run 的正式产品底座”。

## 20. 成功标准

短期成功标准：

- 所有运行都进入 thread。
- 所有状态变化都能从 JSONL replay。
- 所有工具调用都有 ToolCall record。
- 风险动作不会绕过 approval。
- coding 上下文来自 deterministic code search。

中期成功标准：

- `forger code`、`forger run`、TUI 共用同一 runtime。
- skill 能约束工具和输出。
- artifact 成为后续上下文的一等来源。
- memory 有 candidate / promoted 治理。

长期成功标准：

- 用户把 Forger 当作持续工作的 agent workspace。
- coding、research、docs、browser、external systems 都是同一产品模型下的 workflow。
- 任意 thread 都可恢复、可审计、可解释、可继续。
