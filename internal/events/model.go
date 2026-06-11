package events

import (
	"encoding/json"
	"time"
)

// VersionCurrent 是当前 session JSONL event envelope 的版本号。
const VersionCurrent = 1

// Kind 表示 session event 的类型。
type Kind string

const (
	KindThreadCreated    Kind = "thread.created"
	KindMessageAppended  Kind = "message.appended"
	KindRunCreated       Kind = "run.created"
	KindRunStatusChanged Kind = "run.status_changed"
	KindToolCallCreated  Kind = "toolcall.created"
	KindToolCallUpdated  Kind = "toolcall.updated"
	KindArtifactCreated  Kind = "artifact.created"
)

// Envelope 是 sessions/<thread-id>.jsonl 的统一事件包装格式。
type Envelope struct {
	Version   int             `json:"version"`
	Kind      Kind            `json:"kind"`
	ThreadID  string          `json:"thread_id"`
	CreatedAt time.Time       `json:"created_at"`
	Payload   json.RawMessage `json:"payload"`
}
