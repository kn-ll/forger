# Forger

Forger is a local-first agent workbench built around threads, tools, skills, memory, artifacts, approvals, subagents, and Codex-style observability.

This project is a clean rewrite of the Forger product direction. It starts with product-layer primitives first, then adds agent runtime, coding workflows, MCP, skills, memory, RAG, and monitoring incrementally.

Current implemented phases:

- Phase 1: formal thread file layer and event replay
- Phase 2: runtime skeleton and internal run lifecycle
- Phase 3: tool runtime with read-only built-in tools and audited tool calls

## Quick Start

```bash
go test ./...
go run ./cmd/forger thread new "Investigate failing tests"
go run ./cmd/forger thread list
```

## Current Scope

The current code intentionally implements only:

- Thread and run models
- Runtime skeleton and placeholder backend
- Tool call and approval models
- Tool runtime with read-only built-in tools
- Artifact model and file-backed artifact manager
- Monitoring remains design-only and is not yet implemented
- JSON file-backed thread store with replay for messages, runs, tool calls, and artifacts
- Fixed hybrid storage rules: files/JSONL for transcripts, SQLite split into `state.sqlite`, `logs.sqlite`, and `memories.sqlite`
- Minimal CLI for `thread new/list/show`

It intentionally excludes legacy gateway, scheduler tasks, admin command trees, heavy RAG prep, and standalone code/run orchestration.

## Planning

- [产品设计](doc/PRODUCT.md)
- [线程化 Agent 工作台重新设计 Plan](doc/AGENT_WORKBENCH_PLAN.md)
