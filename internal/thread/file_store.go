package thread

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kn-ll/forger/internal/events"
	"github.com/kn-ll/forger/internal/storage"
)

type FileStore struct {
	mu     sync.Mutex
	layout storage.Layout
}

// indexRecord 是 `session_index.jsonl` 的一行。索引文件只保存 thread 的轻量发现信息，
// 供列表和归档浏览使用。
type indexRecord struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type threadCreatedPayload struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    Status    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type threadUpdatedPayload struct {
	Title     string    `json:"title,omitempty"`
	Status    Status    `json:"status,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type messageAppendedPayload struct {
	Message Message `json:"message"`
}

type runCreatedPayload struct {
	Run Run `json:"run"`
}

type runStatusChangedPayload struct {
	RunID     string    `json:"run_id"`
	Status    RunStatus `json:"status"`
	StartedAt time.Time `json:"started_at,omitempty"`
	EndedAt   time.Time `json:"ended_at,omitempty"`
}

// NextIDState 是文件层的 JSON 轻量状态。它只负责稳定分配 thread 编号，不承载
// transcript 或查询状态。
type NextIDState struct {
	NextID int `json:"next_id"`
}

// NewFileStore 创建一个基于 Forger home 的文件存储。
func NewFileStore(home string) *FileStore {
	return &FileStore{layout: storage.DefaultLayout(home)}
}

// Create 创建 thread，并同步写入 `session_index.jsonl` 和 `sessions/<thread-id>.jsonl`。
func (s *FileStore) Create(ctx context.Context, req CreateRequest) (Thread, error) {
	if err := req.Validate(); err != nil {
		return Thread{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.loadNextIDLocked(ctx)
	if err != nil {
		return Thread{}, err
	}
	state.NextID++
	now := time.Now().UTC()
	item := Thread{
		ID:        fmt.Sprintf("thr-%06d", state.NextID),
		Title:     strings.TrimSpace(req.Title),
		Status:    StatusOpen,
		CreatedAt: now,
		UpdatedAt: now,
	}
	payload := threadCreatedPayload{
		ID:        item.ID,
		Title:     item.Title,
		Status:    item.Status,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
	if err := s.appendSessionEventLocked(item.ID, events.KindThreadCreated, now, payload); err != nil {
		return Thread{}, err
	}
	if err := s.appendIndexLocked(indexRecordFromThread(item)); err != nil {
		return Thread{}, err
	}
	if err := s.saveNextIDLocked(state); err != nil {
		return Thread{}, err
	}
	return item, nil
}

// List 从 `session_index.jsonl` 加载 thread 轻量索引，并按创建时间排序。
func (s *FileStore) List(ctx context.Context) ([]Thread, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index, err := s.loadIndexLocked(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Thread, 0, len(index))
	for _, item := range index {
		out = append(out, Thread{
			ID:        item.ID,
			Title:     item.Title,
			Status:    item.Status,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

// Get 从 `sessions/<thread-id>.jsonl` replay 出完整 thread 状态。
func (s *FileStore) Get(ctx context.Context, id string) (Thread, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadThreadLocked(ctx, id)
}

// AppendMessage 追加一条 transcript 消息，并更新 thread 索引的更新时间。
func (s *FileStore) AppendMessage(ctx context.Context, threadID string, req AppendMessageRequest) (Message, error) {
	if err := req.Validate(); err != nil {
		return Message{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	item, err := s.loadThreadLocked(ctx, threadID)
	if err != nil {
		return Message{}, err
	}
	now := time.Now().UTC()
	msg := Message{
		ID:        nextMessageID(item),
		Role:      req.Role,
		Content:   strings.TrimSpace(req.Content),
		RunID:     req.RunID,
		CreatedAt: now,
	}

	if err := s.appendSessionEventLocked(threadID, events.KindMessageAppended,
		now, messageAppendedPayload{Message: msg}); err != nil {
		return Message{}, err
	}
	item.Messages = append(item.Messages, msg)
	item.UpdatedAt = now
	if err := s.appendSessionEventLocked(threadID, events.KindThreadUpdated, now, threadUpdatedPayload{UpdatedAt: now}); err != nil {
		return Message{}, err
	}
	if err := s.appendIndexLocked(indexRecordFromThread(item)); err != nil {
		return Message{}, err
	}
	return msg, nil
}

// CreateRun 在 thread 中创建一条新 run。
func (s *FileStore) CreateRun(ctx context.Context, threadID string, req CreateRunRequest) (Run, error) {
	if err := req.Validate(); err != nil {
		return Run{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	item, err := s.loadThreadLocked(ctx, threadID)
	if err != nil {
		return Run{}, err
	}
	now := time.Now().UTC()
	run := Run{
		ID:        nextRunID(item),
		ThreadID:  threadID,
		Goal:      strings.TrimSpace(req.Goal),
		Status:    RunPending,
		StartedAt: now,
	}
	if err := s.appendSessionEventLocked(threadID, events.KindRunCreated, now, runCreatedPayload{Run: run}); err != nil {
		return Run{}, err
	}
	item.Runs = append(item.Runs, run)
	item.UpdatedAt = now
	if err := s.appendSessionEventLocked(threadID, events.KindThreadUpdated, now, threadUpdatedPayload{UpdatedAt: now}); err != nil {
		return Run{}, err
	}
	if err := s.appendIndexLocked(indexRecordFromThread(item)); err != nil {
		return Run{}, err
	}
	return run, nil
}

// UpdateRun 更新某个 run 的状态，并把生命周期变更写入 JSONL event stream。
func (s *FileStore) UpdateRun(ctx context.Context, threadID string, runID string, req UpdateRunRequest) (Run, error) {
	if err := req.Validate(); err != nil {
		return Run{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	item, err := s.loadThreadLocked(ctx, threadID)
	if err != nil {
		return Run{}, err
	}
	idx := -1
	for i, run := range item.Runs {
		if run.ID == runID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return Run{}, fmt.Errorf("run not found: %s", runID)
	}
	now := time.Now().UTC()
	run := item.Runs[idx]
	run.Status = req.Status
	if run.StartedAt.IsZero() {
		run.StartedAt = now
	}
	if req.Status == RunSucceeded || req.Status == RunFailed || req.Status == RunCanceled {
		run.EndedAt = now
	}
	payload := runStatusChangedPayload{
		RunID:     run.ID,
		Status:    run.Status,
		StartedAt: run.StartedAt,
		EndedAt:   run.EndedAt,
	}
	if err := s.appendSessionEventLocked(threadID, events.KindRunStatusChanged, now, payload); err != nil {
		return Run{}, err
	}
	item.Runs[idx] = run
	item.UpdatedAt = now
	if err := s.appendSessionEventLocked(threadID, events.KindThreadUpdated, now, threadUpdatedPayload{UpdatedAt: now}); err != nil {
		return Run{}, err
	}
	if err := s.appendIndexLocked(indexRecordFromThread(item)); err != nil {
		return Run{}, err
	}
	return run, nil
}

func (s *FileStore) loadThreadLocked(ctx context.Context, id string) (Thread, error) {
	if err := ctx.Err(); err != nil {
		return Thread{}, err
	}
	file, err := os.Open(s.sessionPath(id))
	if errors.Is(err, os.ErrNotExist) {
		return Thread{}, fmt.Errorf("thread not found: %s", id)
	}
	if err != nil {
		return Thread{}, err
	}
	defer file.Close()

	var item Thread
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var envelope events.Envelope
		if err := json.Unmarshal(scanner.Bytes(), &envelope); err != nil {
			return Thread{}, err
		}
		if envelope.Version != events.VersionCurrent {
			return Thread{}, fmt.Errorf("unsupported event version: %d", envelope.Version)
		}
		if err := applyEvent(&item, envelope); err != nil {
			return Thread{}, err
		}
	}
	if err := scanner.Err(); err != nil {
		return Thread{}, err
	}
	if item.ID == "" {
		return Thread{}, fmt.Errorf("thread not found: %s", id)
	}
	return item, nil
}

func applyEvent(item *Thread, envelope events.Envelope) error {
	switch envelope.Kind {
	case events.KindThreadCreated:
		var payload threadCreatedPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return err
		}
		item.ID = payload.ID
		item.Title = payload.Title
		item.Status = payload.Status
		item.CreatedAt = payload.CreatedAt
		item.UpdatedAt = payload.UpdatedAt
	case events.KindThreadUpdated:
		var payload threadUpdatedPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return err
		}
		if payload.Title != "" {
			item.Title = payload.Title
		}
		if payload.Status != "" {
			item.Status = payload.Status
		}
		item.UpdatedAt = payload.UpdatedAt
	case events.KindMessageAppended:
		var payload messageAppendedPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return err
		}
		item.Messages = append(item.Messages, payload.Message)
		if item.UpdatedAt.Before(payload.Message.CreatedAt) {
			item.UpdatedAt = payload.Message.CreatedAt
		}
	case events.KindRunCreated:
		var payload runCreatedPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return err
		}
		item.Runs = append(item.Runs, payload.Run)
		if item.UpdatedAt.Before(payload.Run.StartedAt) {
			item.UpdatedAt = payload.Run.StartedAt
		}
	case events.KindRunStatusChanged:
		var payload runStatusChangedPayload
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			return err
		}
		for i := range item.Runs {
			if item.Runs[i].ID != payload.RunID {
				continue
			}
			item.Runs[i].Status = payload.Status
			if !payload.StartedAt.IsZero() {
				item.Runs[i].StartedAt = payload.StartedAt
			}
			if !payload.EndedAt.IsZero() {
				item.Runs[i].EndedAt = payload.EndedAt
			}
			if item.UpdatedAt.Before(envelope.CreatedAt) {
				item.UpdatedAt = envelope.CreatedAt
			}
			return nil
		}
		return fmt.Errorf("run not found while replaying: %s", payload.RunID)
	default:
		return fmt.Errorf("unsupported event kind: %s", envelope.Kind)
	}
	return nil
}

// loadNextIDLocked 在调用方持有锁时读取 `next_id.json`。
func (s *FileStore) loadNextIDLocked(ctx context.Context) (NextIDState, error) {
	if err := ctx.Err(); err != nil {
		return NextIDState{}, err
	}
	data, err := os.ReadFile(s.nextIDPath())
	if errors.Is(err, os.ErrNotExist) {
		return NextIDState{}, nil
	}
	if err != nil {
		return NextIDState{}, err
	}
	var state NextIDState
	if err := json.Unmarshal(data, &state); err != nil {
		return NextIDState{}, err
	}
	return state, nil
}

// saveNextIDLocked 在调用方持有锁时写入 `next_id.json`，并使用临时文件加 rename
// 降低半写入风险。
func (s *FileStore) saveNextIDLocked(state NextIDState) error {
	path := s.nextIDPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// loadIndexLocked 读取 `session_index.jsonl` 并以最后一条记录作为 thread 当前索引状态。
func (s *FileStore) loadIndexLocked(ctx context.Context) (map[string]indexRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	file, err := os.Open(s.layout.SessionIndexPath)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]indexRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	index := map[string]indexRecord{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record indexRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, err
		}
		index[record.ID] = record
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return index, nil
}

// appendIndexLocked 以 JSONL 形式追加一条索引记录。
func (s *FileStore) appendIndexLocked(record indexRecord) error {
	if err := os.MkdirAll(filepath.Dir(s.layout.SessionIndexPath), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(s.layout.SessionIndexPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = file.Write(append(data, '\n'))
	return err
}

func (s *FileStore) appendSessionEventLocked(threadID string, kind events.Kind, at time.Time, payload any) error {
	path := s.sessionPath(threadID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	envelope := events.Envelope{
		Version:   events.VersionCurrent,
		Kind:      kind,
		ThreadID:  threadID,
		CreatedAt: at,
		Payload:   rawPayload,
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	_, err = file.Write(append(data, '\n'))
	return err
}

func indexRecordFromThread(item Thread) indexRecord {
	return indexRecord{
		ID:        item.ID,
		Title:     item.Title,
		Status:    item.Status,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func nextMessageID(item Thread) string {
	return fmt.Sprintf("msg-%06d", len(item.Messages)+1)
}

func nextRunID(item Thread) string {
	return fmt.Sprintf("run-%06d", len(item.Runs)+1)
}

func (s *FileStore) nextIDPath() string {
	return s.layout.NextIDPath
}

func (s *FileStore) sessionPath(id string) string {
	return filepath.Join(s.layout.SessionsDir, id+".jsonl")
}
