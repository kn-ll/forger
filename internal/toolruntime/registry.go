package toolruntime

import (
	"context"
	"fmt"

	"github.com/kn-ll/forger/internal/tools"
)

// Tool 定义 Forger tool runtime 的最小工具接口。
type Tool interface {
	Spec() tools.Spec
	Run(context.Context, CallRequest) (CallResult, error)
}

// CallRequest 是工具执行输入。
type CallRequest struct {
	ThreadID string
	RunID    string
	Input    map[string]any
}

// CallResult 是工具执行输出。
type CallResult struct {
	Output string
}

// Registry 管理可用工具。
type Registry struct {
	tools map[string]Tool
}

// NewRegistry 创建空 registry。
func NewRegistry() *Registry {
	return &Registry{tools: map[string]Tool{}}
}

// Register 注册工具。
func (r *Registry) Register(tool Tool) {
	r.tools[tool.Spec().Name] = tool
}

// Resolve 查找工具。
func (r *Registry) Resolve(name string) (Tool, error) {
	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool, nil
}
