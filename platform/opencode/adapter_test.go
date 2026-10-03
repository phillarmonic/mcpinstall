package opencode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/phillarmonic/mcpinstall/platform"
)

// isolateEnv clears the variables configDir reads so a developer's real
// configuration directory is never the one a test resolves.
func isolateEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", "")
}

func projectTarget(root string) platform.Target {
	return platform.Target{Scope: platform.ScopeProject, Root: root}
}

func globalTarget(home string) platform.Target {
	return platform.Target{Scope: platform.ScopeGlobal, Root: home}
}

func readMCP(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("%s is not valid JSON: %v", path, err)
	}
	servers, _ := document["mcp"].(map[string]any)
	return servers
}

func TestProjectScopeWritesOpenCodeJSON(t *testing.T) {
	isolateEnv(t)
	root := t.TempDir()

	created, warnings, err := Adapter{}.Install(projectTarget(root), testOptions)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	path := filepath.Join(root, "opencode.json")
	if len(created) != 1 || created[0] != path {
		t.Fatalf("created = %v, want [%s]", created, path)
	}

	entry, ok := readMCP(t, path)[testOptions.ServerName].(map[string]any)
	if !ok {
		t.Fatalf("%s has no %s entry", path, testOptions.ServerName)
	}
	// OpenCode discriminates local from remote servers by "type", and takes the
	// command and its arguments as one array.
	if entry["type"] != "local" {
		t.Errorf("type = %v, want local", entry["type"])
	}
	command, ok := entry["command"].([]any)
	if !ok || len(command) != 1 || command[0] != testOptions.Command {
		t.Errorf("command = %#v, want [%s]", entry["command"], testOptions.Command)
	}
	// OpenCode names its server table "mcp"; the other hosts' keys must not appear.
	if _, present := entry["args"]; present {
		t.Errorf("the entry carries the other hosts' args key: %#v", entry)
	}
}

func TestGlobalScopeResolvesUnderTheConfigDirectory(t *testing.T) {
	isolateEnv(t)
	home := t.TempDir()

	created, _, err := Adapter{}.Install(globalTarget(home), testOptions)
	if err != nil {
		t.Fatal(err)
	}
	// OpenCode's user-level default filename is opencode.jsonc, unlike the
	// project default.
	path := filepath.Join(home, ".config", "opencode", "opencode.jsonc")
	if len(created) != 1 || created[0] != path {
		t.Fatalf("created = %v, want [%s]", created, path)
	}
	if entry, ok := readMCP(t, path)[testOptions.ServerName].(map[string]any); !ok || entry["type"] != "local" {
		t.Fatalf("no local %s entry in %s", testOptions.ServerName, path)
	}
}

func TestGlobalScopeFollowsXDGConfigHome(t *testing.T) {
	home := t.TempDir()
	configHome := filepath.Join(home, "elsewhere")
	t.Setenv("XDG_CONFIG_HOME", configHome)

	created, _, err := Adapter{}.Install(globalTarget(home), testOptions)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(configHome, "opencode", "opencode.jsonc")
	if len(created) != 1 || created[0] != path {
		t.Fatalf("created = %v, want [%s]", created, path)
	}
}

// A configuration file OpenCode would load has to be the one that gets the
// entry: writing a second filename would leave the host reading the first.
func TestInstallMergesIntoTheFileOpenCodeLoads(t *testing.T) {
	isolateEnv(t)
	root := t.TempDir()
	existing := filepath.Join(root, "opencode.jsonc")
	if err := os.WriteFile(existing, []byte(`{"model":"kimi-code/k3"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	created, _, err := Adapter{}.Install(projectTarget(root), testOptions)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 1 || created[0] != existing {
		t.Fatalf("created = %v, want the file OpenCode already loads: [%s]", created, existing)
	}
	if _, err := os.Stat(filepath.Join(root, "opencode.json")); !os.IsNotExist(err) {
		t.Errorf("a second configuration file was created at opencode.json")
	}
	servers := readMCP(t, existing)
	if _, ok := servers[testOptions.ServerName]; !ok {
		t.Errorf("entry missing from the loaded file: %v", servers)
	}
}

// The nested project location counts as a configuration OpenCode loads.
func TestInstallPrefersAnExistingNestedProjectFile(t *testing.T) {
	isolateEnv(t)
	root := t.TempDir()
	nested := filepath.Join(root, ".opencode", "opencode.json")
	if err := os.MkdirAll(filepath.Dir(nested), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nested, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	created, _, err := Adapter{}.Install(projectTarget(root), testOptions)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 1 || created[0] != nested {
		t.Fatalf("created = %v, want [%s]", created, nested)
	}
}

func TestInstallIsIdempotent(t *testing.T) {
	isolateEnv(t)
	root := t.TempDir()
	if _, _, err := (Adapter{}).Install(projectTarget(root), testOptions); err != nil {
		t.Fatal(err)
	}
	created, _, err := Adapter{}.Install(projectTarget(root), testOptions)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 0 {
		t.Fatalf("second install reported changes: %v", created)
	}
}

func TestInstallPreservesUnrelatedSettings(t *testing.T) {
	isolateEnv(t)
	root := t.TempDir()
	path := filepath.Join(root, "opencode.json")
	initial := `{"model":"kimi-code/k3","mcp":{"playwright":{"type":"local","command":["npx","-y","@playwright/mcp"]}}}`
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, _, err := (Adapter{}).Install(projectTarget(root), testOptions); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if document["model"] != "kimi-code/k3" {
		t.Errorf("unrelated setting was dropped: %s", data)
	}
	servers, _ := document["mcp"].(map[string]any)
	if _, ok := servers["playwright"]; !ok {
		t.Errorf("unrelated MCP server was dropped: %s", data)
	}
	if _, ok := servers[testOptions.ServerName]; !ok {
		t.Errorf("managed entry missing: %s", data)
	}
}

// OpenCode has only ever used the "mcp" key, so an unrelated "mcpServers"
// object is another tool's and must be left alone rather than migrated.
func TestInstallLeavesTheOtherHostsKeyAlone(t *testing.T) {
	isolateEnv(t)
	root := t.TempDir()
	path := filepath.Join(root, "opencode.json")
	initial := `{"mcpServers":{"other":{"command":"other"}}}`
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, _, err := (Adapter{}).Install(projectTarget(root), testOptions); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if _, ok := document["mcpServers"]; !ok {
		t.Errorf("the unrelated mcpServers object was removed: %s", data)
	}
}

func TestRemoveDeletesEmptiedFile(t *testing.T) {
	isolateEnv(t)
	root := t.TempDir()
	path := filepath.Join(root, "opencode.json")
	if _, _, err := (Adapter{}).Install(projectTarget(root), testOptions); err != nil {
		t.Fatal(err)
	}

	removed, _, err := Adapter{}.Remove(projectTarget(root), testOptions)
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 1 || removed[0] != path {
		t.Fatalf("removed = %v, want [%s]", removed, path)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("emptied configuration was not deleted: %v", err)
	}
}

func TestRemoveKeepsOtherServers(t *testing.T) {
	isolateEnv(t)
	root := t.TempDir()
	path := filepath.Join(root, "opencode.json")
	initial := `{"mcp":{"other":{"type":"local","command":["other"]}}}`
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := (Adapter{}).Install(projectTarget(root), testOptions); err != nil {
		t.Fatal(err)
	}

	if _, _, err := (Adapter{}).Remove(projectTarget(root), testOptions); err != nil {
		t.Fatal(err)
	}
	servers := readMCP(t, path)
	if _, ok := servers["other"]; !ok {
		t.Errorf("unrelated server was removed: %v", servers)
	}
	if _, ok := servers[testOptions.ServerName]; ok {
		t.Errorf("managed entry survived removal: %v", servers)
	}
}

func TestStatusReportsTheOpenCodeServerTable(t *testing.T) {
	status := Adapter{}.Status()
	if !status.Supported {
		t.Error("opencode should be reported as supported")
	}
	if !status.Capabilities.MCP || status.Capabilities.Format != "json" || status.Capabilities.ServersKey != "mcp" {
		t.Errorf("unexpected capabilities: %#v", status.Capabilities)
	}
	if (Adapter{}).Name() != "opencode" {
		t.Errorf("unexpected name: %s", (Adapter{}).Name())
	}
}
