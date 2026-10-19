package codex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

// readEntry loads the managed server table from a Codex configuration.
func readEntry(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	cfg := map[string]any{}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	servers, ok := cfg["mcp_servers"].(map[string]any)
	if !ok {
		t.Fatalf("%s has no mcp_servers table", path)
	}
	entry, ok := servers[testOptions.ServerName].(map[string]any)
	if !ok {
		t.Fatalf("%s has no managed entry", path)
	}
	return entry
}

func TestInstallWritesTOMLConfig(t *testing.T) {
	root := t.TempDir()

	created, warnings, err := Adapter{}.Install(
		platform.Target{Scope: platform.ScopeProject, Root: root},
		testOptions,
	)
	if err != nil || len(warnings) != 0 {
		t.Fatalf("Install: created=%v warnings=%v err=%v", created, warnings, err)
	}

	path := filepath.Join(root, ".codex", "config.toml")
	if len(created) != 1 || created[0] != path {
		t.Fatalf("created = %v, want [%s]", created, path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
}

func TestInstallSetsApprovalMode(t *testing.T) {
	root := t.TempDir()

	if _, _, err := (Adapter{}).Install(platform.Target{Scope: platform.ScopeProject, Root: root}, testOptions); err != nil {
		t.Fatalf("Install: %v", err)
	}

	entry := readEntry(t, filepath.Join(root, ".codex", "config.toml"))
	if entry["command"] != testOptions.Command {
		t.Fatalf("command = %v, want %v", entry["command"], testOptions.Command)
	}
	// Without this Codex would prompt on every MCP server tool call.
	if entry["default_tools_approval_mode"] != "approve" {
		t.Fatalf("default_tools_approval_mode = %v, want approve", entry["default_tools_approval_mode"])
	}
}

func TestInstallPreservesExistingConfig(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("model = \"gpt-5\"\n"), 0o644); err != nil {
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
	if err := toml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cfg["model"] != "gpt-5" {
		t.Fatal("an unrelated setting was dropped")
	}
}

func TestInstallGlobalHonoursCodexHome(t *testing.T) {
	home := t.TempDir()
	codexHome := filepath.Join(home, "codex-home")
	t.Setenv("CODEX_HOME", codexHome)

	created, _, err := Adapter{}.Install(
		platform.Target{Scope: platform.ScopeGlobal, Root: home},
		testOptions,
	)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	want := filepath.Join(codexHome, "config.toml")
	if len(created) != 1 || created[0] != want {
		t.Fatalf("created = %v, want [%s]", created, want)
	}
}

func TestInstallHTTPWritesHTTPHeaders(t *testing.T) {
	root := t.TempDir()
	opts := common.Options{
		ServerName: "test-server",
		Transport:  common.TransportHTTP,
		URL:        "https://example.test/mcp",
		Headers:    map[string]string{"Authorization": "Bearer token"},
	}
	if _, _, err := (Adapter{}).Install(platform.Target{Scope: platform.ScopeProject, Root: root}, opts); err != nil {
		t.Fatalf("Install: %v", err)
	}
	entry := readEntry(t, filepath.Join(root, ".codex", "config.toml"))
	if entry["url"] != "https://example.test/mcp" {
		t.Fatalf("url = %v", entry["url"])
	}
	headers, ok := entry["http_headers"].(map[string]any)
	if !ok || headers["Authorization"] != "Bearer token" {
		t.Fatalf("http_headers = %#v", entry["http_headers"])
	}
	if _, ok := entry["bearer_token_env_var"]; ok {
		t.Fatal("a literal header install should not name an env var")
	}
}

func TestInstallHTTPCanNameTheTokenVariable(t *testing.T) {
	root := t.TempDir()
	opts := common.Options{
		ServerName:  "test-server",
		Transport:   common.TransportHTTP,
		URL:         "https://example.test/mcp",
		TokenEnvVar: "DEMO_TOKEN",
		Headers:     map[string]string{"Authorization": "Bearer token"},
	}
	if _, _, err := (Adapter{}).Install(platform.Target{Scope: platform.ScopeProject, Root: root}, opts); err != nil {
		t.Fatalf("Install: %v", err)
	}
	entry := readEntry(t, filepath.Join(root, ".codex", "config.toml"))
	if entry["bearer_token_env_var"] != "DEMO_TOKEN" {
		t.Fatalf("bearer_token_env_var = %v", entry["bearer_token_env_var"])
	}
	if _, ok := entry["http_headers"]; ok {
		t.Fatal("the token was copied into the file as well")
	}
}

func TestRemoveDeletesManagedTable(t *testing.T) {
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeProject, Root: root}
	if _, _, err := (Adapter{}).Install(target, testOptions); err != nil {
		t.Fatalf("Install: %v", err)
	}

	removed, _, err := Adapter{}.Remove(target, testOptions)
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(removed) != 1 {
		t.Fatalf("removed = %v", removed)
	}
	if _, err := os.Stat(filepath.Join(root, ".codex", "config.toml")); !os.IsNotExist(err) {
		t.Fatal("configuration file survived removal")
	}
}
