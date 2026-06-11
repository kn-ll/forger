package artifacts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kn-ll/forger/internal/storage"
)

// Manager 管理 artifact 文件落盘。
type Manager struct {
	layout storage.Layout
}

// CreateInput 是写入 artifact 文件层的输入。
type CreateInput struct {
	ThreadID string
	RunID    string
	Kind     Kind
	Title    string
	Content  string
	Metadata map[string]string
}

// NewManager 创建 artifact manager。
func NewManager(home string) *Manager {
	return &Manager{layout: storage.DefaultLayout(home)}
}

// Create 把 artifact 内容写入固定目录结构，并返回 artifact 元数据。
func (m *Manager) Create(ctx context.Context, input CreateInput) (Artifact, error) {
	if err := ctx.Err(); err != nil {
		return Artifact{}, err
	}
	if strings.TrimSpace(input.ThreadID) == "" {
		return Artifact{}, fmt.Errorf("thread id is required")
	}
	if strings.TrimSpace(input.RunID) == "" {
		return Artifact{}, fmt.Errorf("run id is required")
	}
	if strings.TrimSpace(input.Title) == "" {
		return Artifact{}, fmt.Errorf("artifact title is required")
	}
	sum := sha256.Sum256([]byte(input.Content))
	// 先用内容哈希生成稳定 artifact 目录名，便于去重和后续审计比对。
	// TODO: 增加正式 dedup/index 策略；当前只用内容哈希命名，还没有跨 run 引用复用层。
	artifactID := "art-" + hex.EncodeToString(sum[:6])
	dir := filepath.Join(m.layout.ArtifactsDir, input.ThreadID, artifactID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Artifact{}, err
	}
	contentPath := filepath.Join(dir, "content.txt")
	if err := os.WriteFile(contentPath, []byte(input.Content), 0o644); err != nil {
		return Artifact{}, err
	}
	metadata := map[string]string{
		"content_sha256": hex.EncodeToString(sum[:]),
	}
	for k, v := range input.Metadata {
		metadata[k] = v
	}
	metaBytes, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return Artifact{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), append(metaBytes, '\n'), 0o644); err != nil {
		return Artifact{}, err
	}
	return Artifact{
		ID:       artifactID,
		ThreadID: input.ThreadID,
		RunID:    input.RunID,
		Kind:     input.Kind,
		Title:    input.Title,
		URI:      contentPath,
		Metadata: metadata,
	}, nil
}
