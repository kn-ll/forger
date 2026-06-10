package thread

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kn-ll/forger/internal/events"
)

func TestFileStoreReplaysThreadFromJSONLEvents(t *testing.T) {
	home := filepath.Join(t.TempDir(), ".forger")
	store := NewFileStore(home)

	created, err := store.Create(context.Background(), CreateRequest{Title: "Research MCP connectors"})
	if err != nil {
		t.Fatal(err)
	}
	msg, err := store.AppendMessage(context.Background(), created.ID, AppendMessageRequest{
		Role:    RoleUser,
		Content: "Inspect MCP tool registry",
	})
	if err != nil {
		t.Fatal(err)
	}
	run, err := store.CreateRun(context.Background(), created.ID, CreateRunRequest{Goal: "Build replay support"})
	if err != nil {
		t.Fatal(err)
	}
	run, err = store.UpdateRun(context.Background(), created.ID, run.ID, UpdateRunRequest{Status: RunSucceeded})
	if err != nil {
		t.Fatal(err)
	}

	reloaded := NewFileStore(home)
	got, err := reloaded.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != created.Title || got.ID != created.ID {
		t.Fatalf("thread = %+v", got)
	}
	if len(got.Messages) != 1 || got.Messages[0].ID != msg.ID || got.Messages[0].Content != msg.Content {
		t.Fatalf("messages = %+v", got.Messages)
	}
	if len(got.Runs) != 1 || got.Runs[0].ID != run.ID || got.Runs[0].Status != RunSucceeded {
		t.Fatalf("runs = %+v", got.Runs)
	}
	if got.UpdatedAt.Before(got.Messages[0].CreatedAt) {
		t.Fatalf("updated_at was not advanced: %+v", got)
	}
}

func TestFileStoreMaintainsIndexSessionAndNextIDConsistency(t *testing.T) {
	home := filepath.Join(t.TempDir(), ".forger")
	store := NewFileStore(home)

	first, err := store.Create(context.Background(), CreateRequest{Title: "First"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Create(context.Background(), CreateRequest{Title: "Second"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendMessage(context.Background(), second.ID, AppendMessageRequest{
		Role:    RoleAgent,
		Content: "Thread is active",
	}); err != nil {
		t.Fatal(err)
	}

	nextIDBytes, err := os.ReadFile(filepath.Join(home, "next_id.json"))
	if err != nil {
		t.Fatal(err)
	}
	var nextIDState NextIDState
	if err := json.Unmarshal(nextIDBytes, &nextIDState); err != nil {
		t.Fatal(err)
	}
	if nextIDState.NextID != 2 {
		t.Fatalf("next id = %+v", nextIDState)
	}

	indexBytes, err := os.ReadFile(filepath.Join(home, "session_index.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(indexBytes)), "\n")
	if len(lines) != 3 {
		t.Fatalf("index lines = %d, want 3", len(lines))
	}
	var last indexRecord
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &last); err != nil {
		t.Fatal(err)
	}
	if last.ID != second.ID || last.Title != second.Title {
		t.Fatalf("last index record = %+v", last)
	}

	sessionBytes, err := os.ReadFile(filepath.Join(home, "sessions", second.ID+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	sessionLines := strings.Split(strings.TrimSpace(string(sessionBytes)), "\n")
	if len(sessionLines) != 3 {
		t.Fatalf("session lines = %d, want 3", len(sessionLines))
	}
	var firstEnvelope events.Envelope
	if err := json.Unmarshal([]byte(sessionLines[0]), &firstEnvelope); err != nil {
		t.Fatal(err)
	}
	if firstEnvelope.Kind != events.KindThreadCreated || firstEnvelope.Version != events.VersionCurrent {
		t.Fatalf("first envelope = %+v", firstEnvelope)
	}

	list, err := NewFileStore(home).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != first.ID || list[1].ID != second.ID {
		t.Fatalf("list = %+v", list)
	}
}

func TestFileStoreCreateRequiresTitle(t *testing.T) {
	home := filepath.Join(t.TempDir(), ".forger")
	_, err := NewFileStore(home).Create(context.Background(), CreateRequest{})
	if err != ErrTitleRequired {
		t.Fatalf("err = %v", err)
	}
}

func TestFileStoreAppendMessageAndRunValidation(t *testing.T) {
	home := filepath.Join(t.TempDir(), ".forger")
	store := NewFileStore(home)
	created, err := store.Create(context.Background(), CreateRequest{Title: "Validation"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendMessage(context.Background(), created.ID, AppendMessageRequest{}); err != ErrMessageRoleRequired {
		t.Fatalf("append err = %v", err)
	}
	if _, err := store.CreateRun(context.Background(), created.ID, CreateRunRequest{}); err != ErrRunGoalRequired {
		t.Fatalf("create run err = %v", err)
	}
	run, err := store.CreateRun(context.Background(), created.ID, CreateRunRequest{Goal: "Goal"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateRun(context.Background(), created.ID, run.ID, UpdateRunRequest{}); err != ErrRunStatusRequired {
		t.Fatalf("update run err = %v", err)
	}
}
