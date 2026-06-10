# Forger

Forger is a local-first agent workbench built around threads, tools, skills, memory, artifacts, approvals, subagents, and monitoring.

This project is a clean rewrite of the Forger product direction. It starts with product-layer primitives first, then adds agent runtime, coding workflows, MCP, skills, memory, RAG, and monitoring incrementally.

## Quick Start

```bash
go test ./...
go run ./cmd/forger thread new "Investigate failing tests"
go run ./cmd/forger thread list
```

## Current Scope

The current code intentionally implements only:

- Thread and run models
- Tool call and approval models
- Artifact model
- Monitoring event model
- In-memory and JSON file-backed thread stores
- Fixed hybrid storage rules: files/JSONL for transcripts, SQLite split into `state.sqlite`, `logs.sqlite`, and `memories.sqlite`
- Minimal CLI for creating and listing threads

It intentionally excludes legacy gateway, scheduler tasks, admin command trees, heavy RAG prep, and standalone code/run orchestration.

## Planning

- [产品设计](doc/PRODUCT.md)
- [线程化 Agent 工作台重新设计 Plan](doc/AGENT_WORKBENCH_PLAN.md)
