# Phase 1：补齐正式 Thread 文件层

对应总计划：[AGENT_WORKBENCH_PLAN.md](./AGENT_WORKBENCH_PLAN.md)

## 目标

让 thread 文件层完整符合正式存储规则，并可稳定恢复。

## 状态

已完成。

## 范围

1. 增加 `internal/events`。
2. 定义 session JSONL event envelope。
3. 扩展 `thread.Store`：append message、create run、update run。
4. 实现 event replay。
5. 增加 `thread show` CLI。
6. 补齐 `session_index.jsonl`、`next_id.json`、`sessions/<thread-id>.jsonl` 一致性测试。

## 验收结果

- JSONL 能 replay 出完整 thread。
- thread 支持追加消息和 run。
- 现有 `thread new/list` 继续工作。

## 实现了什么

### 1. 统一 session event envelope

新增 `internal/events`，定义 `sessions/<thread-id>.jsonl` 的统一包装格式：

- `version`
- `kind`
- `thread_id`
- `created_at`
- `payload`

当前已实现的 event kind：

- `thread.created`
- `thread.updated`
- `message.appended`
- `run.created`
- `run.status_changed`

对应代码：

- `internal/events/model.go`

### 2. Thread 文件层改为 append-only event stream

`sessions/<thread-id>.jsonl` 不再保存单条 snapshot，而是只追加事件。

这样做有两个目的：

- 文件层成为 thread 的事实源，便于恢复和审计。
- 后续继续增加 tool call、approval、artifact 事件时，不需要推翻存储模型。

对应代码：

- `internal/thread/file_store.go`

### 3. `thread.Store` 扩展为正式 Phase 1 接口

新增以下能力：

- `AppendMessage`
- `CreateRun`
- `UpdateRun`

这样 thread 就不再只是“能创建、能列出”的静态对象，而是能承载 transcript 和 run lifecycle 的真实会话容器。

对应代码：

- `internal/thread/store.go`
- `internal/thread/model.go`

### 4. 基于 event replay 重建 thread

`Get` 现在会逐行读取 `sessions/<thread-id>.jsonl`，按 event kind 把 payload 应用到内存中的 `Thread` 聚合对象，恢复出：

- `title`
- `status`
- `messages`
- `runs`
- `updated_at`

这意味着读取 thread 时不再依赖最后一条 snapshot，而是依赖正式的 event stream replay。

对应代码：

- `internal/thread/file_store.go`

### 5. 新增 `thread show` CLI

增加：

```bash
forger thread show <thread-id>
```

当前会展示：

- thread 基本信息
- message 列表
- run 列表

它直接依赖 replay 后的 `Thread` 结果，保证 CLI 视图和底层恢复逻辑一致。

对应代码：

- `cmd/forger/main.go`

## 如何实现

### 轻量索引层：`session_index.jsonl`

`session_index.jsonl` 继续只承担轻量发现职责，保存：

- `id`
- `title`
- `status`
- `created_at`
- `updated_at`

thread 发生可见变更时，会追加一条新的索引记录；读取时按 thread ID 取最后一条作为当前索引状态。

### 事实层：`sessions/<thread-id>.jsonl`

`sessions/<thread-id>.jsonl` 作为 thread 文件层事实源，只追加 event，不做原位覆盖。

典型写入路径：

1. `Create` 写入 `thread.created`
2. `AppendMessage` 写入 `message.appended`
3. `CreateRun` 写入 `run.created`
4. `UpdateRun` 写入 `run.status_changed`
5. thread 可见状态变更后补写 `thread.updated`

### replay 模型

读取 thread 时：

1. 打开 `sessions/<thread-id>.jsonl`
2. 逐行解析 event envelope
3. 按 `kind` 反序列化 `payload`
4. 将 payload 应用到内存中的 `Thread`
5. 输出聚合后的完整 thread

这个模型保证：

- 文件可恢复
- 事件可审计
- 未来可继续扩展更多 event kind

### ID 与写入稳定性

- `next_id.json` 只负责稳定分配 thread ID。
- `next_id.json` 使用临时文件 + rename，降低半写入风险。
- message ID 和 run ID 当前在 thread 内按顺序生成，满足 Phase 1 的稳定恢复要求。

## 测试覆盖

已补齐以下测试：

- replay 能从 JSONL event stream 恢复完整 thread
- `session_index.jsonl`、`next_id.json`、`sessions/<thread-id>.jsonl` 的一致性
- thread/create/message/run 的基础校验
- `thread show` 的基本可用性

对应代码：

- `internal/thread/file_store_test.go`
- `cmd/forger/main_test.go`

## 当前限制

- 还没有 `internal/runtime`，因此 run 只是文件层生命周期对象，还没有统一执行入口。
- 还没有 tool call、approval、artifact 的 session event。
- 还没有 SQLite 查询层写入器，当前 Phase 1 只完成文件层事实源。

这些内容属于后续 phase，不属于本阶段范围。
