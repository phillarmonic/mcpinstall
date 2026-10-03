package cursor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/phillarmonic/mcpinstall/platform"
)

func TestInstallWritesCursorFile(t *testing.T) {
	root := t.TempDir()

	created, warnings, err := Adapter{}.Install(
		platform.Target{Scope: platform.ScopeProject, Root: root},
		testOptions,
	)
	if err != nil || len(warnings) != 0 {
		t.Fatalf("Install: created=%v warnings=%v err=%v", created, warnings, err)
	}

	path := filepath.Join(root, ".cursor", "mcp.json")
	if len(created) != 1 || created[0] != path {
		t.Fatalf("created = %v, want [%s]", created, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	cfg := map[string]any{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	servers, ok := cfg["mcpServers"].(map[string]any)
	if !ok {
		t.Fatal("mcpServers object is missing")
	}
	if _, ok := servers[testOptions.ServerName]; !ok {
		t.Fatal("managed server was not written")
	}
}

func TestGlobalScopeResolvesUnderTheHomeRoot(t *testing.T) {
	root := t.TempDir()

	created, _, err := Adapter{}.Install(
		platform.Target{Scope: platform.ScopeGlobal, Root: root},
		testOptions,
	)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	path := filepath.Join(root, ".cursor", "mcp.json")
	if len(created) != 1 || created[0] != path {
		t.Fatalf("created = %v, want [%s]", created, path)
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
	if _, err := os.Stat(filepath.Join(root, ".cursor", "mcp.json")); !os.IsNotExist(err) {
		t.Fatal("configuration file survived removal")
	}
}
