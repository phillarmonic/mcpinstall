package common

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// readTOML loads a TOML config for assertions.
func readTOML(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	cfg := map[string]any{}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return cfg
}

func TestUpsertTOMLCreatesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".codex", "config.toml")

	changed, warning, err := UpsertTOMLMCPServer(path, "test settings", "mcp_servers", testServerName, testManagedServer(), Options{})
	if err != nil || warning != "" {
		t.Fatalf("upsert: changed=%v warning=%q err=%v", changed, warning, err)
	}
	if !changed {
		t.Fatal("expected the file to be reported as created")
	}

	servers, ok := readTOML(t, path)["mcp_servers"].(map[string]any)
	if !ok {
		t.Fatal("mcp_servers table is missing")
	}
	entry, ok := servers[testServerName].(map[string]any)
	if !ok {
		t.Fatalf("managed entry is missing: %#v", servers)
	}
	if entry["command"] != testCommand {
		t.Fatalf("command = %v, want %v", entry["command"], testCommand)
	}
}

func TestUpsertTOMLPreservesUnrelatedTables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	original := "model = \"gpt-5\"\n\n[mcp_servers.other]\ncommand = \"other\"\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := UpsertTOMLMCPServer(path, "test settings", "mcp_servers", testServerName, testManagedServer(), Options{}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	cfg := readTOML(t, path)
	if cfg["model"] != "gpt-5" {
		t.Fatalf("unrelated setting was dropped: %#v", cfg["model"])
	}
	servers := cfg["mcp_servers"].(map[string]any)
	if _, ok := servers["other"]; !ok {
		t.Fatal("an unrelated server was dropped")
	}
	if _, ok := servers[testServerName]; !ok {
		t.Fatal("managed server was not added")
	}
}

func TestUpsertTOMLPreservesMalformedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	original := "this is not = = toml"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, warning, err := UpsertTOMLMCPServer(path, "test settings", "mcp_servers", testServerName, testManagedServer(), Options{})
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

func TestUpsertTOMLLeavesExistingEntryAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	original := "[mcp_servers.test-server]\ncommand = \"/custom/path\"\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, warning, err := UpsertTOMLMCPServer(path, "test settings", "mcp_servers", testServerName, testManagedServer(), Options{})
	if err != nil || warning != "" {
		t.Fatalf("upsert: changed=%v warning=%q err=%v", changed, warning, err)
	}
	if changed {
		t.Fatal("a user-customised entry must not be rewritten without force")
	}
}

func TestUpsertTOMLForceReplacesEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	original := "[mcp_servers.test-server]\ncommand = \"/custom/path\"\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, _, err := UpsertTOMLMCPServer(path, "test settings", "mcp_servers", testServerName, testManagedServer(), Options{Force: true})
	if err != nil || !changed {
		t.Fatalf("forced upsert: changed=%v err=%v", changed, err)
	}

	servers := readTOML(t, path)["mcp_servers"].(map[string]any)
	entry := servers[testServerName].(map[string]any)
	if entry["command"] != testCommand {
		t.Fatalf("entry was not replaced: %#v", entry)
	}
}

func TestUpsertTOMLIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")

	if changed, _, err := UpsertTOMLMCPServer(path, "test settings", "mcp_servers", testServerName, testManagedServer(), Options{}); !changed || err != nil {
		t.Fatalf("first upsert: changed=%v err=%v", changed, err)
	}
	changed, warning, err := UpsertTOMLMCPServer(path, "test settings", "mcp_servers", testServerName, testManagedServer(), Options{})
	if err != nil || warning != "" {
		t.Fatalf("second upsert: changed=%v warning=%q err=%v", changed, warning, err)
	}
	if changed {
		t.Fatal("a repeat install reported a change")
	}
}

func TestRemoveTOMLDeletesEmptiedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	original := "[mcp_servers.test-server]\ncommand = \"x\"\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, _, err := RemoveTOMLMCPServer(path, "test settings", "mcp_servers", testServerName, Options{})
	if err != nil || !changed {
		t.Fatalf("remove: changed=%v err=%v", changed, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("emptied config file was left behind at %s", path)
	}
}

func TestRemoveTOMLKeepsOtherTables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	original := "model = \"gpt-5\"\n\n[mcp_servers.test-server]\ncommand = \"x\"\n\n[mcp_servers.other]\ncommand = \"y\"\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := RemoveTOMLMCPServer(path, "test settings", "mcp_servers", testServerName, Options{}); err != nil {
		t.Fatalf("remove: %v", err)
	}

	cfg := readTOML(t, path)
	if cfg["model"] != "gpt-5" {
		t.Fatalf("unrelated setting was dropped: %#v", cfg["model"])
	}
	servers := cfg["mcp_servers"].(map[string]any)
	if _, ok := servers[testServerName]; ok {
		t.Fatal("managed server survived removal")
	}
	if _, ok := servers["other"]; !ok {
		t.Fatal("an unrelated server was removed")
	}
}

func TestRemoveTOMLPreservesMalformedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	original := "not = = toml"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, warning, err := RemoveTOMLMCPServer(path, "test settings", "mcp_servers", testServerName, Options{})
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

func TestToStringMapAcceptsDecoderShapes(t *testing.T) {
	object := map[string]any{"a": 1}
	if got, ok := toStringMap(object); !ok || !reflect.DeepEqual(got, object) {
		t.Fatalf("toStringMap(map[string]any) = %#v, %v", got, ok)
	}
	if _, ok := toStringMap("scalar"); ok {
		t.Fatal("toStringMap accepted a non-table value")
	}
}
