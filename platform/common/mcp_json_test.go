package common

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// readJSON loads a config file for assertions.
func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	cfg := map[string]any{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return cfg
}

// serversOf returns the named server table from a config.
func serversOf(t *testing.T, cfg map[string]any, key string) map[string]any {
	t.Helper()
	servers, ok := cfg[key].(map[string]any)
	if !ok {
		t.Fatalf("config has no %q object: %#v", key, cfg)
	}
	return servers
}

func TestUpsertCreatesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".mcp.json")

	changed, warning, err := UpsertMCPServer(path, "test settings", testServerName, testManagedServer(), Options{})
	if err != nil || warning != "" {
		t.Fatalf("upsert: changed=%v warning=%q err=%v", changed, warning, err)
	}
	if !changed {
		t.Fatal("expected the file to be reported as created")
	}

	server := serversOf(t, readJSON(t, path), "mcpServers")[testServerName]
	want := map[string]any{"command": testCommand, "args": []any{}}
	if !reflect.DeepEqual(server, want) {
		t.Fatalf("server entry = %#v, want %#v", server, want)
	}
}

func TestUpsertPreservesUnrelatedSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".mcp.json")
	original := `{"mcpServers":{"otherServer":{"command":"other"}},"unrelated":{"keep":true}}`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := UpsertMCPServer(path, "test settings", testServerName, testManagedServer(), Options{}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	cfg := readJSON(t, path)
	if _, ok := cfg["unrelated"]; !ok {
		t.Fatal("unrelated settings were dropped")
	}
	servers := serversOf(t, cfg, "mcpServers")
	if _, ok := servers["otherServer"]; !ok {
		t.Fatal("an unrelated MCP server was dropped")
	}
	if _, ok := servers[testServerName]; !ok {
		t.Fatal("managed server was not added")
	}
}

func TestUpsertPreservesMalformedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".mcp.json")
	original := "{ this is not json"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, warning, err := UpsertMCPServer(path, "test settings", testServerName, testManagedServer(), Options{})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if changed || warning == "" {
		t.Fatalf("expected an unchanged result with a warning, got changed=%v warning=%q", changed, warning)
	}

	got, err := os.ReadFile(path)
	if err != nil || string(got) != original {
		t.Fatalf("malformed file was modified: %q, %v", got, err)
	}
}

func TestUpsertLeavesExistingEntryAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".mcp.json")
	original := `{"mcpServers":{"test-server":{"command":"/custom/path"}}}`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, warning, err := UpsertMCPServer(path, "test settings", testServerName, testManagedServer(), Options{})
	if err != nil || warning != "" {
		t.Fatalf("upsert: changed=%v warning=%q err=%v", changed, warning, err)
	}
	if changed {
		t.Fatal("a user-customised entry must not be rewritten without force")
	}

	got, err := os.ReadFile(path)
	if err != nil || string(got) != original {
		t.Fatalf("file changed: %q, %v", got, err)
	}
}

func TestUpsertForceReplacesExistingEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".mcp.json")
	original := `{"mcpServers":{"test-server":{"command":"/custom/path"}}}`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, _, err := UpsertMCPServer(path, "test settings", testServerName, testManagedServer(), Options{Force: true})
	if err != nil || !changed {
		t.Fatalf("forced upsert: changed=%v err=%v", changed, err)
	}

	server := serversOf(t, readJSON(t, path), "mcpServers")[testServerName]
	if server.(map[string]any)["command"] != testCommand {
		t.Fatalf("entry was not replaced: %#v", server)
	}
}

func TestUpsertIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".mcp.json")

	if changed, _, err := UpsertMCPServer(path, "test settings", testServerName, testManagedServer(), Options{}); !changed || err != nil {
		t.Fatalf("first upsert: changed=%v err=%v", changed, err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	changed, warning, err := UpsertMCPServer(path, "test settings", testServerName, testManagedServer(), Options{})
	if err != nil || warning != "" {
		t.Fatalf("second upsert: changed=%v warning=%q err=%v", changed, warning, err)
	}
	if changed {
		t.Fatal("a repeat install reported a change")
	}
	second, err := os.ReadFile(path)
	if err != nil || string(first) != string(second) {
		t.Fatal("a repeat install rewrote the file")
	}
}

func TestUpsertDryRunWritesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".mcp.json")

	changed, _, err := UpsertMCPServer(path, "test settings", testServerName, testManagedServer(), Options{DryRun: true})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if !changed {
		t.Fatal("a dry run must still report the pending change")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("dry run created %s", path)
	}
}

func TestUpsertMigratesLegacyKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")
	original := `{"mcpServers":{"existing":{"command":"other"}}}`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, _, err := UpsertMCPServerInKey(path, "test settings", "servers", testServerName, testManagedServer(), Options{})
	if err != nil || !changed {
		t.Fatalf("upsert: changed=%v err=%v", changed, err)
	}

	cfg := readJSON(t, path)
	if _, ok := cfg["mcpServers"]; ok {
		t.Fatal("the legacy key survived the migration")
	}
	servers := serversOf(t, cfg, "servers")
	if _, ok := servers["existing"]; !ok {
		t.Fatal("a server under the legacy key was dropped")
	}
	if _, ok := servers[testServerName]; !ok {
		t.Fatal("managed server was not added")
	}
}

func TestUpsertInKeyOnlyLeavesLegacyKeyAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")
	original := `{"mcpServers":{"stray":{"command":"other"}}}`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, _, err := UpsertMCPServerInKeyOnly(path, "test settings", "servers", testServerName, testManagedServer(), Options{})
	if err != nil || !changed {
		t.Fatalf("upsert: changed=%v err=%v", changed, err)
	}

	cfg := readJSON(t, path)
	// The stray table must still be there under its own key: renaming it would
	// activate servers the host never read.
	if _, ok := serversOf(t, cfg, "mcpServers")["stray"]; !ok {
		t.Fatal("the strict writer renamed or dropped the legacy key")
	}
	if _, ok := serversOf(t, cfg, "servers")[testServerName]; !ok {
		t.Fatal("managed server was not written to the host's key")
	}
}

func TestRemoveInKeyOnlyLeavesLegacyKeyAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")
	original := `{"mcpServers":{"test-server":{"command":"other"}}}`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, _, err := RemoveMCPServerInKeyOnly(path, "test settings", "servers", testServerName, Options{})
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if changed {
		t.Fatal("the strict writer removed an entry the host never wrote")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Fatalf("the file changed: %s", data)
	}
}

func TestUpsertReplacesLegacyTemplate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mcp.json")
	legacy := map[string]any{"command": "old-binary", "args": []any{"mcp"}}
	original := `{"mcpServers":{"test-server":{"command":"old-binary","args":["mcp"]}}}`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, _, err := UpsertMCPServerInKeyReplacingLegacy(path, "test settings", "mcpServers", testServerName, testManagedServer(), legacy, Options{})
	if err != nil || !changed {
		t.Fatalf("upsert: changed=%v err=%v", changed, err)
	}

	server := serversOf(t, readJSON(t, path), "mcpServers")[testServerName]
	if server.(map[string]any)["command"] != testCommand {
		t.Fatalf("legacy template was not upgraded: %#v", server)
	}
}

func TestUpsertRejectsNonObjectServers(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".mcp.json")
	original := `{"mcpServers":"not an object"}`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, warning, err := UpsertMCPServer(path, "test settings", testServerName, testManagedServer(), Options{})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if changed || warning == "" {
		t.Fatalf("expected an unchanged result with a warning, got changed=%v warning=%q", changed, warning)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != original {
		t.Fatalf("file changed: %q, %v", got, err)
	}
}

func TestRemoveKeepsOtherServers(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".mcp.json")
	original := `{"mcpServers":{"test-server":{"command":"x"},"other":{"command":"y"}}}`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, _, err := RemoveMCPServer(path, "test settings", testServerName, Options{})
	if err != nil || !changed {
		t.Fatalf("remove: changed=%v err=%v", changed, err)
	}

	servers := serversOf(t, readJSON(t, path), "mcpServers")
	if _, ok := servers[testServerName]; ok {
		t.Fatal("managed server survived removal")
	}
	if _, ok := servers["other"]; !ok {
		t.Fatal("an unrelated server was removed")
	}
}

func TestRemoveDeletesEmptiedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".mcp.json")
	original := `{"mcpServers":{"test-server":{"command":"x"}}}`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, _, err := RemoveMCPServer(path, "test settings", testServerName, Options{})
	if err != nil || !changed {
		t.Fatalf("remove: changed=%v err=%v", changed, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("emptied config file was left behind at %s", path)
	}
}

func TestRemovePreservesUnrelatedKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".mcp.json")
	original := `{"mcpServers":{"test-server":{"command":"x"}},"unrelated":true}`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := RemoveMCPServer(path, "test settings", testServerName, Options{}); err != nil {
		t.Fatalf("remove: %v", err)
	}

	cfg := readJSON(t, path)
	if _, ok := cfg["mcpServers"]; ok {
		t.Fatal("the emptied server table was left behind")
	}
	if cfg["unrelated"] != true {
		t.Fatal("an unrelated key was dropped")
	}
}

func TestRemovePreservesMalformedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".mcp.json")
	original := "{ not json"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, warning, err := RemoveMCPServer(path, "test settings", testServerName, Options{})
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if changed || warning == "" {
		t.Fatalf("expected an unchanged result with a warning, got changed=%v warning=%q", changed, warning)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("malformed file was removed: %v", err)
	}
}

func TestRemoveMissingFileIsNoop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.json")

	changed, warning, err := RemoveMCPServer(path, "test settings", testServerName, Options{})
	if err != nil || changed || warning != "" {
		t.Fatalf("remove of a missing file: changed=%v warning=%q err=%v", changed, warning, err)
	}
}

func TestRemoveDryRunKeepsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".mcp.json")
	original := `{"mcpServers":{"test-server":{"command":"x"}}}`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, _, err := RemoveMCPServer(path, "test settings", testServerName, Options{DryRun: true})
	if err != nil || !changed {
		t.Fatalf("dry-run remove: changed=%v err=%v", changed, err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != original {
		t.Fatalf("dry run modified the file: %q, %v", got, err)
	}
}
