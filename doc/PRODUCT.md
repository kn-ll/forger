# Forger Product Design

Forger is a thread-centered agent workbench. Coding is one workflow, not the whole product.

## Product Shape

```text
Forger = Thread + Tools + Skills + Memory + Artifacts + Approval + Subagents + Monitoring
```

## Execution Backend Boundary

Forger keeps its own product kernel. External agent frameworks may be integrated, but they must stay behind Forger runtime boundaries.

Current direction:

- Primary implementation language is Go.
- Eino is the preferred first execution backend for orchestration and agent execution.
- Google ADK is not a near-term dependency for the main runtime path.

The boundary is strict:

- Forger owns `Thread`, `Message`, `Run`, `ToolCall`, `ApprovalRequest`, `Artifact`, and session event schemas.
- Forger owns file-layer truth, SQLite query models, approval policy, audit, and artifact tracking.
- External frameworks only provide execution and orchestration capabilities behind adapters.

This means Eino should be integrated as an execution backend or workflow adapter, not as the product model itself. Forger should not replace its thread/run/session model with framework-native session abstractions.

## First-Class Objects

| Object | Purpose |
|---|---|
| Thread | Long-lived user task container |
| Message | User, agent, system, and tool messages |
| Run | One agent execution inside a thread |
| ToolCall | One tool invocation with input, output, risk, and timing |
| ApprovalRequest | Permission request for risky work |
| Artifact | Reusable output such as diffs, reports, documents, screenshots, and verification logs |
| MonitoringEvent | Queryable operational event for runs, tools, approvals, costs, and failures |

## Monitoring Shape

Forger monitoring should follow a Codex-style observability model, not a traditional infrastructure-dashboard-first model.

It has three layers:

1. Runtime state
   - Thread activity
   - Run timeline
   - Tool call progress
   - Approval wait points
   - Artifact production
2. Audit and governance
   - Approval decisions
   - Risky actions
   - Tool execution traces
   - Compliance and export-oriented logs
3. Analytics
   - Run counts
   - Success and failure rates
   - Tool latency
   - Approval wait time
   - Token and cost trends

The primary product surface is not charts first. The primary surface is:

- thread-level timeline
- run-level detail
- approval and tool trace visibility
- debug and governance queries

Charts and dashboards come after the runtime state and audit layers exist.

## Deleted Legacy Concepts

The clean rewrite does not include these legacy product concepts:

- Gateway
- Scheduler task runner
- Task session runtime
- Admin command tree as the primary UX
- Heavy default RAG preparation
- Standalone code/run orchestration

## Storage Rules

Forger follows a Codex-style hybrid storage model. File storage and SQLite coexist by design.

| Storage | Purpose |
|---|---|
| TOML | User and project configuration |
| JSON | Lightweight state, cache, credentials metadata, and bootstrap state |
| JSONL | Raw thread transcripts and append-only event streams |
| `state.sqlite` | Queryable core product state for threads, runs, tool calls, approvals, and artifacts |
| `logs.sqlite` | Runtime-state events, audit logs, failure categories, retry markers, and analytics statistics |
| `memories.sqlite` | Long-term memory records, tags, scopes, scores, and memory metadata |
| Filesystem directories | Attachments, generated files, screenshots, browser sessions, skill packages, and artifacts |

The file layer keeps durable source material that should remain inspectable and append-friendly. SQLite indexes that material for thread timelines, audit queries, analytics, filtering, and cross-thread queries. One should not replace the other.

Standard layout under `~/.forger`:

```text
~/.forger/config.toml
~/.forger/auth.json
~/.forger/history.jsonl
~/.forger/session_index.jsonl
~/.forger/next_id.json
~/.forger/logs.sqlite
~/.forger/memories.sqlite
~/.forger/state.sqlite
~/.forger/sessions/
~/.forger/artifacts/
~/.forger/attachments/
```

Responsibilities are fixed:

1. `history.jsonl` stores interactive input history, not thread state.
2. `session_index.jsonl` stores lightweight thread and archive discovery records.
3. `next_id.json` stores the next file-layer thread number.
4. `sessions/<thread-id>.jsonl` stores the raw source-of-truth transcript and append-only event stream for that thread.
5. `state.sqlite` stores normalized thread/run/tool/approval/artifact relations for product queries.
6. `logs.sqlite` stores runtime-state events, audit, failure, retry, and analytics query data.
7. `memories.sqlite` stores long-term memory records and lookup metadata.
8. Artifact binaries and generated files stay in filesystem directories, never inside SQLite blobs.
9. Configuration stays in TOML; lightweight bootstrap and credential metadata stay in JSON.

## Milestones

1. Product primitives and in-memory store
2. Hybrid storage: JSONL transcripts plus split SQLite state/logs/memories
3. Tool registry, approvals, artifacts
4. Monitoring platform
5. Agent runtime with pluggable backend boundary
6. Coding workflow and deterministic code search
7. Skills, memory, RAG, MCP, and subagents
