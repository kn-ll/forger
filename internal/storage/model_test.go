package storage

import "testing"

func TestDefaultLayout(t *testing.T) {
	layout := DefaultLayout("/tmp/forger-home")
	if layout.ConfigPath != "/tmp/forger-home/config.toml" {
		t.Fatalf("config path = %q", layout.ConfigPath)
	}
	if layout.AuthPath != "/tmp/forger-home/auth.json" {
		t.Fatalf("auth path = %q", layout.AuthPath)
	}
	if layout.HistoryPath != "/tmp/forger-home/history.jsonl" {
		t.Fatalf("history path = %q", layout.HistoryPath)
	}
	if layout.SessionIndexPath != "/tmp/forger-home/session_index.jsonl" {
		t.Fatalf("session index path = %q", layout.SessionIndexPath)
	}
	if layout.NextIDPath != "/tmp/forger-home/next_id.json" {
		t.Fatalf("next id path = %q", layout.NextIDPath)
	}
	if layout.StateDBPath != "/tmp/forger-home/state.sqlite" {
		t.Fatalf("state db path = %q", layout.StateDBPath)
	}
	if layout.LogsDBPath != "/tmp/forger-home/logs.sqlite" {
		t.Fatalf("logs db path = %q", layout.LogsDBPath)
	}
	if layout.MemoriesDBPath != "/tmp/forger-home/memories.sqlite" {
		t.Fatalf("memories db path = %q", layout.MemoriesDBPath)
	}
	if layout.SessionsDir != "/tmp/forger-home/sessions" {
		t.Fatalf("sessions dir = %q", layout.SessionsDir)
	}
}
