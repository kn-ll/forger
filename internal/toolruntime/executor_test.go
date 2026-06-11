package toolruntime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kn-ll/forger/internal/artifacts"
	"github.com/kn-ll/forger/internal/thread"
)

func TestExecutorWritesToolCallAndArtifact(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "README.md"), []byte("alpha\nbeta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(workspace, ".forger")
	store := thread.NewFileStore(home)
	created, err := store.Create(context.Background(), thread.CreateRequest{Title: "Inspect workspace"})
	if err != nil {
		t.Fatal(err)
	}
	run, err := store.CreateRun(context.Background(), created.ID, thread.CreateRunRequest{Goal: "inspect"})
	if err != nil {
		t.Fatal(err)
	}
	registry := NewRegistry()
	RegisterBuiltins(registry, workspace)
	executor := NewExecutor(registry, store, artifacts.NewManager(home))

	call, artifactsOut, err := executor.Execute(context.Background(), created.ID, run.ID, "file.read", map[string]any{"path": "README.md"})
	if err != nil {
		t.Fatal(err)
	}
	if call.Status != "succeeded" || !strings.Contains(call.Output, "alpha") {
		t.Fatalf("call = %+v", call)
	}
	if len(artifactsOut) != 1 {
		t.Fatalf("artifacts = %+v", artifactsOut)
	}
	got, err := store.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ToolCalls) != 1 || len(got.Artifacts) != 1 {
		t.Fatalf("thread = %+v", got)
	}
}
