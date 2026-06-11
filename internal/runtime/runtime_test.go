package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kn-ll/forger/internal/agentbackend"
	"github.com/kn-ll/forger/internal/artifacts"
	"github.com/kn-ll/forger/internal/thread"
	"github.com/kn-ll/forger/internal/toolruntime"
)

func TestStartRunCreatesThreadRunMessagesAndToolCalls(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "README.md"), []byte("hello\nworld\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(workspace, ".forger")
	store := thread.NewFileStore(home)
	registry := toolruntime.NewRegistry()
	toolruntime.RegisterBuiltins(registry, workspace)
	executor := toolruntime.NewExecutor(registry, store, artifacts.NewManager(home))
	rt := New(store, agentbackend.PlaceholderRunner{}, executor)
	created, err := store.Create(context.Background(), thread.CreateRequest{Title: "Existing thread"})
	if err != nil {
		t.Fatal(err)
	}

	item, run, err := rt.StartRun(context.Background(), StartRunRequest{
		ThreadID: created.ID,
		Goal:     "fix code search",
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != thread.RunSucceeded {
		t.Fatalf("run = %+v", run)
	}
	if len(item.Messages) == 0 {
		t.Fatalf("messages = %+v", item.Messages)
	}
	last := item.Messages[len(item.Messages)-1]
	if last.Role != thread.RoleAgent || !strings.Contains(last.Content, "placeholder agent response") {
		t.Fatalf("last message = %+v", last)
	}
	if len(item.ToolCalls) == 0 || item.ToolCalls[0].Tool != "file.tree" {
		t.Fatalf("tool calls = %+v", item.ToolCalls)
	}
	if len(item.Artifacts) == 0 {
		t.Fatalf("artifacts = %+v", item.Artifacts)
	}
	if _, err := os.Stat(item.Artifacts[0].URI); err != nil {
		t.Fatalf("artifact file missing: %v", err)
	}
}

func TestStartRunRequiresExistingThread(t *testing.T) {
	workspace := t.TempDir()
	home := filepath.Join(workspace, ".forger")
	store := thread.NewFileStore(home)
	registry := toolruntime.NewRegistry()
	toolruntime.RegisterBuiltins(registry, workspace)
	executor := toolruntime.NewExecutor(registry, store, artifacts.NewManager(home))
	rt := New(store, agentbackend.PlaceholderRunner{}, executor)

	if _, _, err := rt.StartRun(context.Background(), StartRunRequest{Goal: "fix code search"}); err == nil || !strings.Contains(err.Error(), "thread id is required") {
		t.Fatalf("err = %v", err)
	}
}
