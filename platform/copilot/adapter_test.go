package copilot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/phillarmonic/mcpinstall/platform"
)

// readServers loads the VS Code server table from a configuration file.
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
	servers, ok := cfg["servers"].(map[string]any)
	if !ok {
		t.Fatalf("%s has no servers object", path)
	}
	return servers
}

func TestInstallWritesVSCodeFile(t *testing.T) {
	root := t.TempDir()

	created, warnings, err := Adapter{}.Install(
		platform.Target{Scope: platform.ScopeProject, Root: root},
		testOptions,
	)
	if err != nil || len(warnings) != 0 {
		t.Fatalf("Install: created=%v warnings=%v err=%v", created, warnings, err)
	}

	path := filepath.Join(root, ".vscode", "mcp.json")
	if len(created) != 1 || created[0] != path {
		t.Fatalf("created = %v, want [%s]", created, path)
	}
	if _, ok := readServers(t, path)[testOptions.ServerName]; !ok {
		t.Fatal("managed server was not written")
	}
}

func TestInstallMigratesLegacyKey(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".vscode", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"mcpServers":{"other":{"command":"x"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := (Adapter{}).Install(platform.Target{Scope: platform.ScopeProject, Root: root}, testOptions); err != nil {
		t.Fatalf("Install: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := map[string]any{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := cfg["mcpServers"]; ok {
		t.Fatal("the legacy key survived the migration")
	}
	servers, ok := cfg["servers"].(map[string]any)
	if !ok {
		t.Fatal("servers object is missing")
	}
	if _, ok := servers["other"]; !ok {
		t.Fatal("a server under the legacy key was dropped")
	}
}

func TestGlobalScopeResolvesUnderTheVSUserDir(t *testing.T) {
	// The profile directory is derived from the environment on Linux and
	// Windows, so clear it: without this the test would write into the
	// developer's real VS Code configuration and still pass, because it asserts
	// against the same helper the adapter uses.
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("APPDATA", "")

	root := t.TempDir()

	created, _, err := Adapter{}.Install(
		platform.Target{Scope: platform.ScopeGlobal, Root: root},
		testOptions,
	)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("created = %v", created)
	}

	// The profile directory sits outside the home root on some platforms, so
	// assert on the shape rather than one hard-coded location.
	want := filepath.Join(vscodeUserDir(root), "mcp.json")
	if created[0] != want {
		t.Fatalf("created = %v, want [%s]", created, want)
	}
}

func TestRemoveDeletesEmptiedFile(t *testing.T) {
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeProject, Root: root}
	if _, _, err := (Adapter{}).Install(target, testOptions); err != nil {
		t.Fatalf("Install: %v", err)
	}

	if _, _, err := (Adapter{}).Remove(target, testOptions); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".vscode", "mcp.json")); !os.IsNotExist(err) {
		t.Fatal("configuration file survived removal")
	}
}
