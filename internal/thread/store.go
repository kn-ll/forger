package thread

import (
	"context"
	"errors"

	"github.com/kn-ll/forger/internal/artifacts"
	"github.com/kn-ll/forger/internal/tools"
)

// ErrTitleRequired 表示创建 thread 时缺少标题。
var ErrTitleRequired = errors.New("thread title is required")
var ErrMessageRoleRequired = errors.New("message role is required")
var ErrMessageContentRequired = errors.New("message content is required")
var ErrRunGoalRequired = errors.New("run goal is required")
var ErrRunStatusRequired = errors.New("run status is required")

// Store 定义 thread 持久化边界。CLI、TUI 和后续 agent runtime 都依赖这个接口，
// 而不是直接依赖具体存储实现。
type Store interface {
	Create(context.Context, CreateRequest) (Thread, error)
	List(context.Context) ([]Thread, error)
	Get(context.Context, string) (Thread, error)
	AppendMessage(context.Context, string, AppendMessageRequest) (Message, error)
	CreateRun(context.Context, string, CreateRunRequest) (Run, error)
	UpdateRun(context.Context, string, string, UpdateRunRequest) (Run, error)
	AppendToolCall(context.Context, string, tools.Call) (tools.Call, error)
	AppendArtifact(context.Context, string, artifacts.Artifact) (artifacts.Artifact, error)
}
