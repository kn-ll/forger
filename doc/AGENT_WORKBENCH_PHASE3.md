# Phase 3：Tool Runtime 和只读工具

对应总计划：[AGENT_WORKBENCH_PLAN.md](./AGENT_WORKBENCH_PLAN.md)

## 目标

统一工具注册、执行、审计。

## 状态

已完成。

## 范围

1. 新增 `internal/toolruntime`。
2. 实现 registry。
3. 实现 `file.tree`、`file.read`、`search.rg`、`git.diff`、`git.log`。
4. ToolCall 写入 event stream。
5. Tool output 可写 artifact。

## 验收结果

- 每个工具调用都有 ToolCall record。
- 只读 coding context 可通过工具获得。
- 工具失败有结构化错误。

## 实现了什么

### 1. 新增 `toolruntime.Registry`

新增 `internal/toolruntime`，提供工具注册与查找能力。

当前工具 runtime 的最小抽象包括：

- `Tool`
- `CallRequest`
- `CallResult`
- `Registry`
- `Executor`

对应代码：

- `internal/toolruntime/registry.go`
- `internal/toolruntime/executor.go`

### 2. 实现首批只读工具

已实现：

- `file.tree`
- `file.read`
- `search.rg`
- `git.diff`
- `git.log`

其中：

- `file.tree` 用于快速浏览工作区结构
- `file.read` 用于读取文件内容
- `search.rg` 用于基于 `rg` 做确定性文本检索
- `git.diff` / `git.log` 用于读取 git 上下文

对应代码：

- `internal/toolruntime/builtin.go`

### 3. ToolCall 进入 session event stream

扩展了 session event kinds：

- `toolcall.created`
- `toolcall.updated`
- `artifact.created`

thread 文件层现在不仅能 replay message 和 run，也能 replay：

- `ToolCalls`
- `Artifacts`

对应代码：

- `internal/events/model.go`
- `internal/thread/file_store.go`
- `internal/thread/model.go`
- `internal/thread/store.go`

### 4. 新增 artifact manager

工具输出现在会通过 `internal/artifacts.Manager` 写入：

```text
artifacts/<thread-id>/<artifact-id>/
  content.txt
  metadata.json
```

当前 executor 会把工具输出写成 artifact，并回写 thread 的 artifact event。

对应代码：

- `internal/artifacts/manager.go`

### 5. runtime 集成只读工具

`Phase 2` 的 `PlaceholderRunner` 现在在目标看起来像代码任务时，会主动调用 `file.tree`。这让 runtime 在没有真实 LLM 的情况下，仍然能留下真实的 ToolCall 与 artifact 审计记录。

## 如何实现

### Registry + Executor 分层

`Registry` 只负责注册和解析工具；`Executor` 负责：

1. 创建 `ToolCall` running record
2. 执行工具
3. 更新 `ToolCall` 为 succeeded / failed
4. 把输出写成 artifact
5. 把 `ToolCall` 和 artifact 回写到 thread event stream

这样工具定义与持久化/审计逻辑分离，后续增加 approval、sandbox 时也能继续沿用。

### ToolCall 的两阶段事件

一次成功工具调用会经历：

1. `toolcall.created`
2. `artifact.created`
3. `toolcall.updated`

失败工具调用会经历：

1. `toolcall.created`
2. `toolcall.updated`

这样 replay 时既能恢复工具是否被计划执行，也能恢复最终状态。

### 工具输出作为 artifact

当前设计不把工具原始输出只留在 `ToolCall.Output`，而是同步写文件层 artifact。这样后续：

- UI 可以直接展示 artifact
- 大输出不会只能依赖 event stream 回看
- trace、审计、复用边界更清晰

## 测试覆盖

- executor 能写入 ToolCall 与 artifact
- thread replay 能恢复 ToolCall 与 artifact
- `StartRun` 路径上能生成真实 `file.tree` ToolCall

对应代码：

- `internal/toolruntime/executor_test.go`
- `internal/thread/file_store_test.go`
- `internal/runtime/runtime_test.go`
