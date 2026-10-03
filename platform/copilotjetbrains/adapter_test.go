package copilotjetbrains

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/phillarmonic/mcpinstall/platform"
)

// isolateEnv clears the variables the plugin's directory resolution consults, so
// a developer's XDG_CONFIG_HOME or LOCALAPPDATA can never point a test at their
// real Copilot configuration.
func isolateEnv(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("LOCALAPPDATA", "")
}

// pluginPath is where a run rooted at root is expected to write.
func pluginPath(root string) string {
	return filepath.Join(configDir(root), pluginDir, "mcp.json")
}

// globalTarget builds the only target this host accepts.
func globalTarget(root string) platform.Target {
	return platform.Target{Scope: platform.ScopeGlobal, Root: root}
}

// readServers loads the host's server table from a configuration file.
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

func TestConfigDirPerPlatform(t *testing.T) {
	const home = "/home/tester"
	env := func(values map[string]string) func(string) string {
		return func(key string) string { return values[key] }
	}

	cases := []struct {
		name string
		goos string
		env  map[string]string
		want string
	}{
		{
			name: "windows prefers LOCALAPPDATA",
			goos: "windows",
			env:  map[string]string{"LOCALAPPDATA": filepath.Join("C:", "Users", "tester", "AppData", "Local")},
			want: filepath.Join("C:", "Users", "tester", "AppData", "Local"),
		},
		{
			name: "windows falls back to the home directory",
			goos: "windows",
			want: filepath.Join(home, "AppData", "Local"),
		},
		{
			name: "darwin prefers XDG_CONFIG_HOME",
			goos: "darwin",
			env:  map[string]string{"XDG_CONFIG_HOME": "/xdg/config"},
			want: "/xdg/config",
		},
		{
			name: "darwin falls back to .config",
			goos: "darwin",
			want: filepath.Join(home, ".config"),
		},
		{
			name: "linux falls back to .config",
			goos: "linux",
			want: filepath.Join(home, ".config"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := configDirFor(tc.goos, home, env(tc.env)); got != tc.want {
				t.Fatalf("configDirFor(%s) = %q, want %q", tc.goos, got, tc.want)
			}
		})
	}
}

func TestProjectScopeIsRefused(t *testing.T) {
	isolateEnv(t)
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeProject, Root: root}

	created, warnings, err := Adapter{}.Install(target, testOptions)
	if err == nil {
		t.Fatalf("Install reported success: created=%v warnings=%v", created, warnings)
	}
	if created != nil || warnings != nil {
		t.Fatalf("a refused install reported work: created=%v warnings=%v", created, warnings)
	}

	// A project-scope run must not reach the user's configuration, and must not
	// leave anything behind in the project either.
	entries, readErr := os.ReadDir(root)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("a refused install wrote %d entries", len(entries))
	}

	if _, _, err := (Adapter{}).Remove(target, testOptions); err == nil {
		t.Fatal("Remove accepted a project-scope target")
	}
}

func TestInstallWritesThePluginFile(t *testing.T) {
	isolateEnv(t)
	root := t.TempDir()

	created, warnings, err := Adapter{}.Install(globalTarget(root), testOptions)
	if err != nil || len(warnings) != 0 {
		t.Fatalf("Install: created=%v warnings=%v err=%v", created, warnings, err)
	}

	path := pluginPath(root)
	if len(created) != 1 || created[0] != path {
		t.Fatalf("created = %v, want [%s]", created, path)
	}
	if _, ok := readServers(t, path)[testOptions.ServerName]; !ok {
		t.Fatal("managed server was not written")
	}
}

func TestInstallIsIdempotent(t *testing.T) {
	isolateEnv(t)
	root := t.TempDir()
	target := globalTarget(root)

	if _, _, err := (Adapter{}).Install(target, testOptions); err != nil {
		t.Fatalf("first install: %v", err)
	}
	created, _, err := (Adapter{}).Install(target, testOptions)
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if len(created) != 0 {
		t.Fatalf("a repeat install reported changes: %v", created)
	}
}

func TestRemoveDeletesEmptiedFile(t *testing.T) {
	isolateEnv(t)
	root := t.TempDir()
	target := globalTarget(root)

	if _, _, err := (Adapter{}).Install(target, testOptions); err != nil {
		t.Fatalf("Install: %v", err)
	}
	removed, _, err := (Adapter{}).Remove(target, testOptions)
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(removed) != 1 {
		t.Fatalf("removed = %v, want the configuration file", removed)
	}
	if _, err := os.Stat(pluginPath(root)); !os.IsNotExist(err) {
		t.Fatal("configuration file survived removal")
	}
}

func TestRemoveKeepsOtherServers(t *testing.T) {
	isolateEnv(t)
	root := t.TempDir()
	path := pluginPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"servers":{"other":{"command":"x"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := (Adapter{}).Install(globalTarget(root), testOptions); err != nil {
		t.Fatalf("Install: %v", err)
	}
	if _, _, err := (Adapter{}).Remove(globalTarget(root), testOptions); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	servers := readServers(t, path)
	if _, ok := servers["other"]; !ok {
		t.Fatal("an unrelated MCP server was dropped")
	}
	if _, ok := servers[testOptions.ServerName]; ok {
		t.Fatal("the managed server survived removal")
	}
}

func TestInstallLeavesLegacyKeyAlone(t *testing.T) {
	isolateEnv(t)
	root := t.TempDir()
	path := pluginPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	// The plugin has only ever read "servers", so a "mcpServers" object is
	// something else's. Renaming it would bring those servers to life.
	original := `{"mcpServers":{"stray":{"command":"other"}}}`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, _, err := (Adapter{}).Install(globalTarget(root), testOptions); err != nil {
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
	stray, ok := cfg["mcpServers"].(map[string]any)
	if !ok {
		t.Fatal("the legacy key was renamed away")
	}
	if _, ok := stray["stray"]; !ok {
		t.Fatal("a server under the legacy key was dropped")
	}
	if _, ok := readServers(t, path)[testOptions.ServerName]; !ok {
		t.Fatal("managed server was not written")
	}
}

func TestRemoveLeavesLegacyKeyAlone(t *testing.T) {
	isolateEnv(t)
	root := t.TempDir()
	path := pluginPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	original := `{"mcpServers":{"aekf":{"command":"other"}}}`
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	removed, _, err := (Adapter{}).Remove(globalTarget(root), testOptions)
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(removed) != 0 {
		t.Fatalf("removed = %v, want nothing: the host never wrote that key", removed)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Fatalf("the file changed: %s", data)
	}
}
