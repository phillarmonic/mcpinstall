package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

// readServers loads the managed entry from a Claude configuration file.
func readServers(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	cfg := map[string]any{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	servers, ok := cfg["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("%s has no mcpServers object", path)
	}
	return servers
}

func TestInstallProjectScopeUsesProjectFile(t *testing.T) {
	root := t.TempDir()

	created, warnings, err := Adapter{}.Install(
		platform.Target{Scope: platform.ScopeProject, Root: root},
		testOptions,
	)
	if err != nil || len(warnings) != 0 {
		t.Fatalf("Install: created=%v warnings=%v err=%v", created, warnings, err)
	}

	path := filepath.Join(root, ".mcp.json")
	if len(created) != 1 || created[0] != path {
		t.Fatalf("created = %v, want [%s]", created, path)
	}
	if _, ok := readServers(t, path)[testOptions.ServerName]; !ok {
		t.Fatal("managed server was not written")
	}
}

func TestInstallGlobalScopeUsesUserFile(t *testing.T) {
	root := t.TempDir()

	created, _, err := Adapter{}.Install(
		platform.Target{Scope: platform.ScopeGlobal, Root: root},
		testOptions,
	)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	path := filepath.Join(root, ".claude.json")
	if len(created) != 1 || created[0] != path {
		t.Fatalf("created = %v, want [%s]", created, path)
	}
	if _, ok := readServers(t, path)[testOptions.ServerName]; !ok {
		t.Fatal("managed server was not written")
	}
}

func TestInstallHonoursCommandOverride(t *testing.T) {
	root := t.TempDir()

	if _, _, err := (Adapter{}).Install(
		platform.Target{Scope: platform.ScopeProject, Root: root},
		common.Options{ServerName: "test-server", Command: "/opt/bin/aekf-mcp"},
	); err != nil {
		t.Fatalf("Install: %v", err)
	}

	entry := readServers(t, filepath.Join(root, ".mcp.json"))[testOptions.ServerName].(map[string]any)
	if entry["command"] != "/opt/bin/aekf-mcp" {
		t.Fatalf("command = %v", entry["command"])
	}
}

func TestInstallPreservesExistingSettings(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".mcp.json")
	if err := os.WriteFile(path, []byte(`{"mcpServers":{"other":{"command":"x"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := (Adapter{}).Install(platform.Target{Scope: platform.ScopeProject, Root: root}, testOptions); err != nil {
		t.Fatalf("Install: %v", err)
	}

	servers := readServers(t, path)
	if _, ok := servers["other"]; !ok {
		t.Fatal("an unrelated server was dropped")
	}
	if _, ok := servers[testOptions.ServerName]; !ok {
		t.Fatal("managed server was not added")
	}
}

func TestRemoveDeletesManagedEntry(t *testing.T) {
	root := t.TempDir()
	if _, _, err := (Adapter{}).Install(platform.Target{Scope: platform.ScopeProject, Root: root}, testOptions); err != nil {
		t.Fatalf("Install: %v", err)
	}

	removed, _, err := Adapter{}.Remove(platform.Target{Scope: platform.ScopeProject, Root: root}, testOptions)
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(removed) != 1 || removed[0] != filepath.Join(root, ".mcp.json") {
		t.Fatalf("removed = %v", removed)
	}
	if _, err := os.Stat(filepath.Join(root, ".mcp.json")); !os.IsNotExist(err) {
		t.Fatal("configuration file survived removal")
	}
}

func TestDryRunWritesNothing(t *testing.T) {
	root := t.TempDir()

	created, _, err := Adapter{}.Install(
		platform.Target{Scope: platform.ScopeProject, Root: root},
		common.Options{ServerName: "test-server", Command: "test-command", DryRun: true},
	)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("dry run reported %v, want the pending path", created)
	}
	if _, err := os.Stat(filepath.Join(root, ".mcp.json")); !os.IsNotExist(err) {
		t.Fatal("dry run wrote a file")
	}
}
