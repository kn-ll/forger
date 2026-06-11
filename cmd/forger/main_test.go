package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThreadShowCLI(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	if code := run(context.Background(), []string{"thread", "new", "Investigate replay"}); code != 0 {
		t.Fatalf("create exit code = %d", code)
	}
	if code := run(context.Background(), []string{"thread", "show", "thr-000001"}); code != 0 {
		t.Fatalf("show exit code = %d", code)
	}

	sessionPath := filepath.Join(tmp, ".forger", "sessions", "thr-000001.jsonl")
	if _, err := os.Stat(sessionPath); err != nil {
		t.Fatalf("session file missing: %v", err)
	}
}

func TestThreadShowCLIOutput(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	stdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		os.Stdout = stdout
	})

	if code := run(context.Background(), []string{"thread", "new", "Investigate replay"}); code != 0 {
		t.Fatalf("create exit code = %d", code)
	}
	if code := run(context.Background(), []string{"thread", "show", "thr-000001"}); code != 0 {
		t.Fatalf("show exit code = %d", code)
	}
	_ = w.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Thread\tthr-000001") || !strings.Contains(out, "Title\tInvestigate replay") {
		t.Fatalf("stdout = %q", out)
	}
}
