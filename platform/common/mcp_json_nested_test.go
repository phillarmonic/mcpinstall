package common

import (
	"os"
	"path/filepath"
	"testing"
)

func nestedConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{
  "mcp": {
    "servers": {
      "other": {"command": "other-mcp"}
    },
    "keep": true
  },
  "unrelated": "value"
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUpsertMCPServerAtPathCreatesTheEntry(t *testing.T) {
	path := nestedConfig(t)
	changed, warning, err := UpsertMCPServerAtPath(
		path, "test settings", []string{"mcp", "servers"}, "demo",
		map[string]any{"command": "demo-mcp", "args": []any{}}, Options{},
	)
	if err != nil || warning != "" {
		t.Fatalf("upsert: changed=%v warning=%q err=%v", changed, warning, err)
	}
	if !changed {
		t.Fatal("expected a change")
	}
	cfg := readJSON(t, path)
	servers, ok := cfg["mcp"].(map[string]any)["servers"].(map[string]any)
	if !ok {
		t.Fatalf("nested servers missing: %#v", cfg)
	}
	if _, ok := servers["other"]; !ok {
		t.Fatal("sibling server was lost")
	}
	if _, ok := servers["demo"]; !ok {
		t.Fatal("new server was not written")
	}
	if cfg["unrelated"] != "value" {
		t.Fatalf("unrelated key was lost: %#v", cfg)
	}
}

func TestUpsertMCPServerAtPathLeavesAnExistingEntryAlone(t *testing.T) {
	path := nestedConfig(t)
	changed, _, err := UpsertMCPServerAtPath(
		path, "test settings", []string{"mcp", "servers"}, "other",
		map[string]any{"command": "replacement"}, Options{},
	)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if changed {
		t.Fatal("an existing entry should be preserved without Force")
	}
	servers := readJSON(t, path)["mcp"].(map[string]any)["servers"].(map[string]any)
	if servers["other"].(map[string]any)["command"] != "other-mcp" {
		t.Fatalf("existing entry was overwritten: %#v", servers["other"])
	}
}

func TestRemoveMCPServerAtPathPrunesTheMap(t *testing.T) {
	path := nestedConfig(t)
	changed, warning, err := RemoveMCPServerAtPath(
		path, "test settings", []string{"mcp", "servers"}, "other", Options{},
	)
	if err != nil || warning != "" {
		t.Fatalf("remove: changed=%v warning=%q err=%v", changed, warning, err)
	}
	if !changed {
		t.Fatal("expected a change")
	}
	cfg := readJSON(t, path)
	mcp, ok := cfg["mcp"].(map[string]any)
	if !ok {
		t.Fatalf("mcp object was removed even though it held a sibling: %#v", cfg)
	}
	if _, ok := mcp["servers"]; ok {
		t.Fatalf("emptied servers map was not pruned: %#v", mcp)
	}
	if mcp["keep"] != true {
		t.Fatalf("sibling key was lost: %#v", mcp)
	}
}

func TestNestedWriterPreservesANonObjectContainer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"mcp": "not an object"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	changed, warning, err := UpsertMCPServerAtPath(
		path, "test settings", []string{"mcp", "servers"}, "demo",
		map[string]any{"command": "demo-mcp"}, Options{},
	)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if changed || warning == "" {
		t.Fatalf("changed=%v warning=%q, want a preserved file with a warning", changed, warning)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"mcp": "not an object"}` {
		t.Fatalf("file was rewritten: %s", data)
	}
}

func TestNestedWritersHonourDryRun(t *testing.T) {
	path := nestedConfig(t)
	opts := Options{DryRun: true}
	changed, _, err := UpsertMCPServerAtPath(
		path, "test settings", []string{"mcp", "servers"}, "demo",
		map[string]any{"command": "demo-mcp"}, opts,
	)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if !changed {
		t.Fatal("a dry run should report the pending change")
	}
	cfg := readJSON(t, path)
	servers := cfg["mcp"].(map[string]any)["servers"].(map[string]any)
	if _, ok := servers["demo"]; ok {
		t.Fatal("a dry run wrote the entry")
	}
}
