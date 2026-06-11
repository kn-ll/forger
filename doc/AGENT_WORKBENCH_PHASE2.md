# Phase 2：RunRuntime 骨架

对应总计划：[AGENT_WORKBENCH_PLAN.md](./AGENT_WORKBENCH_PLAN.md)

## 目标

建立统一运行入口，但暂不接真实模型。

## 状态

已完成。

## 范围

1. 新增 `internal/runtime`。
2. 新增 `internal/agentbackend` 抽象，但先只提供占位 backend。
3. 实现 `StartRun`、`CancelRun`、`ResumeRun` 接口。
4. run 状态写入 event stream。

## 验收结果

- runtime 能创建或复用 thread 并启动一次 run。
- run lifecycle 可在 thread 中查看。
- run event 可审计。

补充说明：

- Phase 2 只建立内部 runtime 能力，不再暴露单次 `forger run` CLI 入口。

## 实现了什么

### 1. 新增 `ThreadRuntime`

新增 `internal/runtime`，提供统一运行入口：

- `StartRun`
- `CancelRun`
- `ResumeRun`

当前 `ResumeRun` 先显式返回未实现，保证 runtime 边界先固定，再进入后续恢复逻辑实现。

对应代码：

- `internal/runtime/runtime.go`

### 2. 新增 `agentbackend` 抽象

新增 `internal/agentbackend`，把外部执行框架和 Forger runtime 解耦。当前先提供：

- `Runner`
- `Request`
- `Emitter`
- `PlaceholderRunner`

`PlaceholderRunner` 用于 Phase 2 验证统一运行链路，不引入真实模型。

对应代码：

- `internal/agentbackend/model.go`
- `internal/agentbackend/placeholder.go`

### 3. 打通 run lifecycle

`StartRun` 现在会：

1. 创建或加载 thread
2. 创建 run
3. 把 run 状态切到 `running`
4. 调用 backend
5. 根据执行结果把 run 标记为 `succeeded` 或 `failed`

这些状态变化都通过 thread 文件层 event stream 持久化。

## 如何实现

### runtime 作为统一入口

`ThreadRuntime` 成为所有执行类能力的收口点，避免 CLI、后续 TUI、future code aliases 各自绕过 thread/run 生命周期直接操作模型。

### backend 只负责执行，不负责产品状态

backend 通过 `Emitter` 与 runtime 交互，不能直接持久化 thread。这样保证：

- run 状态仍由 Forger 控制
- session events 仍由 Forger 定义
- 后续接 Eino 时只需要实现 adapter

### 占位 backend 的作用

Phase 2 不追求真实模型能力，而是先验证：

- runtime 到 backend 的路径
- backend 回写 message 的路径
- run lifecycle 的审计路径

## 测试覆盖

- `StartRun` 会创建 thread、run、agent message

对应代码：

- `internal/runtime/runtime_test.go`
