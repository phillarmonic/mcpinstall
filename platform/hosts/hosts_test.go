package hosts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

// clearEnv removes the relocation variables a developer machine may have set,
// so every test resolves against its temporary root.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA",
		"CLAUDE_CONFIG_DIR", "CODEX_HOME", "GROK_HOME", "KIMI_CODE_HOME",
		"KIRO_HOME", "COPILOT_HOME", "PRIME_AGENT_CODING_AGENT_DIR",
		"PI_CODING_AGENT_DIR", "OMP_PROFILE", "PI_PROFILE", "PI_CONFIG_DIR",
	} {
		t.Setenv(name, "")
	}
}

func stdioOptions() common.Options {
	return common.Options{
		ServerName: "demo",
		Command:    "demo-mcp",
		Args:       []string{"--stdio"},
	}
}

func httpOptions() common.Options {
	return common.Options{
		ServerName: "demo",
		Transport:  common.TransportHTTP,
		URL:        "https://example.test/mcp",
		Headers:    map[string]string{"Authorization": "Bearer token"},
	}
}

// globalPathFor returns the path we expect a host to write at global scope.
func globalPathFor(t *testing.T, host string) string {
	t.Helper()
	switch host {
	case "claude-desktop":
		if runtime.GOOS == "darwin" {
			return filepath.Join("Library", "Application Support", "Claude", "claude_desktop_config.json")
		}
		if runtime.GOOS == "windows" {
			return filepath.Join("AppData", "Roaming", "Claude", "claude_desktop_config.json")
		}
		return filepath.Join(".config", "Claude", "claude_desktop_config.json")
	case "zed":
		if runtime.GOOS == "windows" {
			return filepath.Join("AppData", "Roaming", "Zed", "settings.json")
		}
		return filepath.Join(".config", "zed", "settings.json")
	case "gemini":
		return filepath.Join(".gemini", "settings.json")
	case "openclaw":
		return filepath.Join(".openclaw", "config.json")
	case "omp":
		return filepath.Join(".omp", "agent", "mcp.json")
	case "antigravity":
		return filepath.Join(".gemini", "config", "mcp_config.json")
	case "zero":
		return filepath.Join(".config", "zero", "config.json")
	case "zcode":
		return filepath.Join(".zcode", "cli", "config.json")
	case "devin":
		return filepath.Join(".devin", "config.json")
	case "grok":
		return filepath.Join(".grok", "config.toml")
	case "kimi":
		return filepath.Join(".kimi-code", "mcp.json")
	case "kiro":
		return filepath.Join(".kiro", "settings", "mcp.json")
	case "command-code":
		return filepath.Join(".commandcode", "mcp.json")
	case "ibm-bob":
		return filepath.Join(".bob", "settings", "mcp.json")
	case "copilot-cli":
		return filepath.Join(".copilot", "mcp-config.json")
	case "droid":
		return filepath.Join(".factory", "mcp.json")
	case "muse":
		return filepath.Join(".config", "muse", "settings.json")
	case "prime-agent":
		return filepath.Join(".prime", "agent", "settings.json")
	case "opencode2":
		return filepath.Join(".config", "opencode", "opencode.jsonc")
	default:
		t.Fatalf("no expected path for %q", host)
		return ""
	}
}

// TestGlobalHostsWriteTheirDocumentedFile exercises one real install for every
// global host, so a wrong path or a broken writer is caught here rather than
// only in a shape unit test.
func TestGlobalHostsWriteTheirDocumentedFile(t *testing.T) {
	clearEnv(t)
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeGlobal, Root: root}

	for _, host := range []string{
		"claude-desktop", "gemini", "openclaw", "omp", "antigravity",
		"zero", "zcode", "devin", "grok", "kimi", "kiro", "command-code",
		"ibm-bob", "copilot-cli", "droid", "zed", "muse", "prime-agent",
		"opencode2",
	} {
		t.Run(host, func(t *testing.T) {
			adapter := All()[host]
			opts := stdioOptions()
			if host == "zcode" {
				// ZCode drops command/args entries, so it only accepts HTTP.
				opts = httpOptions()
			}
			paths, warnings, err := adapter.Install(target, opts)
			if err != nil {
				t.Fatalf("Install: %v", err)
			}
			if len(warnings) != 0 {
				t.Fatalf("warnings = %v", warnings)
			}
			want := filepath.Join(root, globalPathFor(t, host))
			if len(paths) != 1 || paths[0] != want {
				t.Fatalf("paths = %v, want [%s]", paths, want)
			}
			if _, err := os.Stat(want); err != nil {
				t.Fatalf("expected %s: %v", want, err)
			}
		})
	}
}

// TestNestedHostsWriteUnderTheServerMap checks the three nested shapes rather
// than only the file they land in.
func TestNestedHostsWriteUnderTheServerMap(t *testing.T) {
	clearEnv(t)
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeGlobal, Root: root}

	cases := []struct {
		host string
		path string
		key  string
	}{
		{"openclaw", globalPathFor(t, "openclaw"), "mcp.servers"},
		{"zero", globalPathFor(t, "zero"), "mcp.servers"},
		{"zcode", globalPathFor(t, "zcode"), "mcp.servers"},
		{"opencode2", globalPathFor(t, "opencode2"), "mcp.servers"},
	}
	for _, tc := range cases {
		t.Run(tc.host, func(t *testing.T) {
			opts := stdioOptions()
			if tc.host == "zcode" {
				opts = httpOptions()
			}
			if _, _, err := All()[tc.host].Install(target, opts); err != nil {
				t.Fatalf("Install: %v", err)
			}
			rootMap := readJSON(t, filepath.Join(root, tc.path))
			value, ok := nestedValue(rootMap, tc.key)
			if !ok {
				t.Fatalf("file has no %s map: %#v", tc.key, rootMap)
			}
			servers, ok := value.(map[string]any)
			if !ok {
				t.Fatalf("%s is not an object: %#v", tc.key, value)
			}
			if _, ok := servers["demo"]; !ok {
				t.Fatalf("%s has no demo entry: %#v", tc.key, servers)
			}
		})
	}
}

func TestStdioEntriesMatchEachHostShape(t *testing.T) {
	opts := stdioOptions()
	cases := []struct {
		host string
		want map[string]any
	}{
		{"gemini", map[string]any{"command": "demo-mcp", "args": []any{"--stdio"}, "timeout": float64(geminiTimeoutMs)}},
		{"antigravity", map[string]any{"command": "demo-mcp", "args": []any{"--stdio"}, "timeout": float64(geminiTimeoutMs)}},
		{"omp", map[string]any{"type": "stdio", "command": "demo-mcp", "args": []any{"--stdio"}, "enabled": true}},
		{"devin", map[string]any{"command": "demo-mcp", "args": []any{"--stdio"}}},
		{"command-code", map[string]any{"transport": "stdio", "enabled": true, "command": "demo-mcp", "args": []any{"--stdio"}}},
		{"copilot-cli", map[string]any{"type": "local", "command": "demo-mcp", "args": []any{"--stdio"}, "tools": []any{"*"}}},
		{"muse", map[string]any{"type": "stdio", "command": "demo-mcp", "args": []any{"--stdio"}, "mode": "optional"}},
		{"openclaw", map[string]any{"command": "demo-mcp", "args": []any{"--stdio"}}},
	}
	for _, tc := range cases {
		t.Run(tc.host, func(t *testing.T) {
			got, err := All()[tc.host].(Descriptor).Entry(opts)
			if err != nil {
				t.Fatalf("Entry: %v", err)
			}
			assertEntry(t, got, tc.want)
		})
	}
}

func TestHTTPEntriesMatchEachHostShape(t *testing.T) {
	opts := httpOptions()
	cases := []struct {
		host string
		want map[string]any
	}{
		{"gemini", map[string]any{"httpUrl": "https://example.test/mcp", "timeout": float64(geminiTimeoutMs), "headers": map[string]any{"Authorization": "Bearer token"}}},
		{"antigravity", map[string]any{"serverUrl": "https://example.test/mcp", "timeout": float64(geminiTimeoutMs), "headers": map[string]any{"Authorization": "Bearer token"}}},
		{"openclaw", map[string]any{"url": "https://example.test/mcp", "transport": "streamable-http", "headers": map[string]any{"Authorization": "Bearer token"}}},
		{"omp", map[string]any{"type": "http", "url": "https://example.test/mcp", "enabled": true, "headers": map[string]any{"Authorization": "Bearer token"}}},
		{"devin", map[string]any{"url": "https://example.test/mcp", "transport": "http", "headers": map[string]any{"Authorization": "Bearer token"}}},
		{"command-code", map[string]any{"transport": "http", "enabled": true, "url": "https://example.test/mcp", "headers": map[string]any{"Authorization": "Bearer token"}}},
		{"ibm-bob", map[string]any{"type": "streamable-http", "url": "https://example.test/mcp", "protocolNegotiation": "legacy", "headers": map[string]any{"Authorization": "Bearer token"}}},
		{"copilot-cli", map[string]any{"type": "http", "url": "https://example.test/mcp", "headers": map[string]any{"Authorization": "Bearer token"}, "tools": []any{"*"}}},
		{"opencode2", map[string]any{"type": "remote", "url": "https://example.test/mcp", "oauth": false, "headers": map[string]any{"Authorization": "Bearer token"}}},
		{"muse", map[string]any{"type": "http", "url": "https://example.test/mcp", "headers": map[string]any{"Authorization": "Bearer token"}, "mode": "optional"}},
	}
	for _, tc := range cases {
		t.Run(tc.host, func(t *testing.T) {
			got, err := All()[tc.host].(Descriptor).Entry(opts)
			if err != nil {
				t.Fatalf("Entry: %v", err)
			}
			assertEntry(t, got, tc.want)
		})
	}
}

func TestTokenEnvIndirectionStaysOutOfTheFile(t *testing.T) {
	opts := httpOptions()
	delete(opts.Headers, "Authorization")
	opts.TokenEnvVar = "DEMO_TOKEN"

	grok, err := All()["grok"].(Descriptor).Entry(opts)
	if err != nil {
		t.Fatalf("grok Entry: %v", err)
	}
	if _, ok := grok["headers"]; ok {
		t.Fatalf("grok wrote a literal header: %#v", grok)
	}

	droid, err := All()["droid"].(Descriptor).Entry(opts)
	if err != nil {
		t.Fatalf("droid Entry: %v", err)
	}
	headers, _ := droid["headers"].(map[string]any)
	if headers["Authorization"] != "Bearer ${DEMO_TOKEN}" {
		t.Fatalf("droid header = %#v, want the variable reference", headers)
	}

	prime, err := All()["prime-agent"].(Descriptor).Entry(opts)
	if err != nil {
		t.Fatalf("prime agent Entry: %v", err)
	}
	if prime["bearerTokenEnvVar"] != "DEMO_TOKEN" {
		t.Fatalf("prime agent entry = %#v", prime)
	}
}

func TestClaudeDesktopBridgesHTTPThroughMcpRemote(t *testing.T) {
	entry, err := All()["claude-desktop"].(Descriptor).Entry(httpOptions())
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	if entry["command"] != "npx" {
		t.Fatalf("command = %v, want npx", entry["command"])
	}
	args, _ := entry["args"].([]any)
	joined := strings.Join(toStrings(args), " ")
	for _, want := range []string{"-y", "mcp-remote", "https://example.test/mcp", "--header", "Authorization: Bearer token"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("args = %q, want %q", joined, want)
		}
	}
}

func TestMuseUsesTheExistingSnakeCaseMapAndSchemaVersion(t *testing.T) {
	clearEnv(t)
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeGlobal, Root: root}
	path := filepath.Join(root, globalPathFor(t, "muse"))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"mcp_servers":{"other":{"command":"other"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, _, err := All()["muse"].Install(target, stdioOptions()); err != nil {
		t.Fatalf("Install: %v", err)
	}
	rootMap := readJSON(t, path)
	if rootMap["schema_version"] != float64(1) {
		t.Fatalf("schema_version = %#v, want 1", rootMap["schema_version"])
	}
	servers, _ := rootMap["mcp_servers"].(map[string]any)
	if _, ok := servers["other"]; !ok {
		t.Fatalf("sibling server was lost: %#v", servers)
	}
	entry, _ := servers["demo"].(map[string]any)
	if entry["transport"] != "stdio" {
		t.Fatalf("legacy map entry = %#v, want transport stdio", entry)
	}
	if _, ok := rootMap["mcpServers"]; ok {
		t.Fatal("Muse must not hold both server maps")
	}
}

func TestMuseRefusesBothServerMaps(t *testing.T) {
	clearEnv(t)
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeGlobal, Root: root}
	path := filepath.Join(root, globalPathFor(t, "muse"))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"mcpServers":{},"mcp_servers":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := All()["muse"].Install(target, stdioOptions()); err == nil {
		t.Fatal("expected an error when both maps are present")
	}
}

func TestInstallIsIdempotentAndRemoveCleansUp(t *testing.T) {
	clearEnv(t)
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeGlobal, Root: root}

	for _, host := range []string{"gemini", "openclaw", "grok", "muse"} {
		t.Run(host, func(t *testing.T) {
			adapter := All()[host]
			if _, _, err := adapter.Install(target, stdioOptions()); err != nil {
				t.Fatalf("first Install: %v", err)
			}
			paths, warnings, err := adapter.Install(target, stdioOptions())
			if err != nil {
				t.Fatalf("second Install: %v", err)
			}
			if len(paths) != 0 || len(warnings) != 0 {
				t.Fatalf("second install changed something: paths=%v warnings=%v", paths, warnings)
			}
			removed, _, err := adapter.Remove(target, stdioOptions())
			if err != nil {
				t.Fatalf("Remove: %v", err)
			}
			if len(removed) == 0 {
				t.Fatal("Remove reported no change")
			}
			if _, err := os.Stat(filepath.Join(root, globalPathFor(t, host))); !os.IsNotExist(err) {
				t.Fatalf("file survived cleanup: %v", err)
			}
		})
	}
}

func TestZCodeSkipsAStdioInstall(t *testing.T) {
	clearEnv(t)
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeGlobal, Root: root}
	paths, warnings, err := All()["zcode"].Install(target, stdioOptions())
	if err != nil {
		t.Fatalf("a stdio request should skip, not fail: %v", err)
	}
	if len(paths) != 0 || len(warnings) != 1 || !strings.Contains(warnings[0], "HTTP") {
		t.Fatalf("paths=%v warnings=%v", paths, warnings)
	}
}

func TestScopeSupportMatchesTheDescriptor(t *testing.T) {
	clearEnv(t)
	globalOnly := []string{"gemini", "openclaw", "zed", "zcode", "droid", "prime-agent"}
	for _, host := range globalOnly {
		adapter, ok := All()[host].(platform.ScopeSupport)
		if !ok {
			t.Fatalf("%s does not implement ScopeSupport", host)
		}
		if adapter.SupportsScope(platform.ScopeProject) {
			t.Fatalf("%s claims project support", host)
		}
		if !adapter.SupportsScope(platform.ScopeGlobal) {
			t.Fatalf("%s does not claim global support", host)
		}
	}
	swival := All()["swival"].(platform.ScopeSupport)
	if swival.SupportsScope(platform.ScopeGlobal) {
		t.Fatal("swival claims global support")
	}
}

func TestSwivalWritesAtTheProjectRoot(t *testing.T) {
	clearEnv(t)
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o750); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(root, "nested", "project")
	if err := os.MkdirAll(child, 0o750); err != nil {
		t.Fatal(err)
	}
	target := platform.Target{Scope: platform.ScopeProject, Root: child}
	paths, _, err := All()["swival"].Install(target, stdioOptions())
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	want := filepath.Join(root, ".swival", "mcp.json")
	if len(paths) != 1 || paths[0] != want {
		t.Fatalf("paths = %v, want [%s]", paths, want)
	}
}

func TestEnvOverridesRelocateTheConfig(t *testing.T) {
	clearEnv(t)
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeGlobal, Root: root}

	cases := []struct {
		env      string
		host     string
		override string
		path     string
	}{
		{"GROK_HOME", "grok", "grok-home", filepath.Join("grok-home", "config.toml")},
		{"KIMI_CODE_HOME", "kimi", "kimi-home", filepath.Join("kimi-home", "mcp.json")},
		{"KIRO_HOME", "kiro", "kiro-home", filepath.Join("kiro-home", "settings", "mcp.json")},
		{"COPILOT_HOME", "copilot-cli", "copilot-home", filepath.Join("copilot-home", "mcp-config.json")},
		{"PRIME_AGENT_CODING_AGENT_DIR", "prime-agent", "prime-home", filepath.Join("prime-home", "settings.json")},
		{"PI_CODING_AGENT_DIR", "omp", "omp-home", filepath.Join("omp-home", "mcp.json")},
	}
	for _, tc := range cases {
		t.Run(tc.host, func(t *testing.T) {
			t.Setenv(tc.env, filepath.Join(root, tc.override))
			paths, _, err := All()[tc.host].Install(target, stdioOptions())
			if err != nil {
				t.Fatalf("Install: %v", err)
			}
			want := filepath.Join(root, tc.path)
			if len(paths) != 1 || paths[0] != want {
				t.Fatalf("paths = %v, want [%s]", paths, want)
			}
		})
	}
}

func TestKiroRejectsAPlainHTTPRemote(t *testing.T) {
	base := common.Options{ServerName: "demo", Transport: common.TransportHTTP}
	for _, tc := range []struct {
		url string
		ok  bool
	}{
		{"https://example.test/mcp", true},
		{"http://localhost:49374/mcp", true},
		{"http://127.0.0.1:49374/mcp", true},
		{"http://example.test/mcp", false},
	} {
		t.Run(tc.url, func(t *testing.T) {
			opts := base
			opts.URL = tc.url
			err := validateKiroURL(opts)
			if tc.ok && err != nil {
				t.Fatalf("validateKiroURL = %v, want nil", err)
			}
			if !tc.ok && err == nil {
				t.Fatal("validateKiroURL = nil, want an error")
			}
		})
	}
}

func TestPrimeAgentRejectsAnUnloadableName(t *testing.T) {
	for _, name := range []string{"demo", "demo-1", "d_1"} {
		if err := validatePrimeAgentName(common.Options{ServerName: name}); err != nil {
			t.Fatalf("validatePrimeAgentName(%q) = %v", name, err)
		}
	}
	for _, name := range []string{"", "-demo", "demo.name", strings.Repeat("a", 65)} {
		if err := validatePrimeAgentName(common.Options{ServerName: name}); err == nil {
			t.Fatalf("validatePrimeAgentName(%q) = nil, want an error", name)
		}
	}
}

func TestOMPProfilePrecedence(t *testing.T) {
	clearEnv(t)
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeGlobal, Root: root}

	t.Setenv("OMP_PROFILE", "work")
	t.Setenv("PI_CODING_AGENT_DIR", filepath.Join(root, "ignored"))
	paths, _, err := All()["omp"].Install(target, stdioOptions())
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	want := filepath.Join(root, ".omp", "profiles", "work", "agent", "mcp.json")
	if len(paths) != 1 || paths[0] != want {
		t.Fatalf("paths = %v, want [%s]", paths, want)
	}

	t.Setenv("OMP_PROFILE", "")
	t.Setenv("PI_CONFIG_DIR", ".omp-alt")
	t.Setenv("PI_CODING_AGENT_DIR", "")
	paths, _, err = All()["omp"].Install(target, stdioOptions())
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	want = filepath.Join(root, ".omp-alt", "agent", "mcp.json")
	if len(paths) != 1 || paths[0] != want {
		t.Fatalf("paths = %v, want [%s]", paths, want)
	}
}

func TestUnsupportedHostsAreNamedButNotSweptIntoAll(t *testing.T) {
	for _, host := range []string{"dsh", "pi"} {
		descriptor, ok := All()[host].(Descriptor)
		if !ok {
			t.Fatalf("%s is not a descriptor", host)
		}
		if descriptor.Status().Supported {
			t.Fatalf("%s reports supported", host)
		}
		if descriptor.Unsupported == "" {
			t.Fatalf("%s has no guidance", host)
		}
		if !descriptor.SupportsScope(platform.ScopeGlobal) {
			t.Fatalf("%s would be skipped before its guidance could be shown", host)
		}
	}
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return value
}

func readTOML(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var value map[string]any
	if err := toml.Unmarshal(data, &value); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return value
}

func nestedValue(root map[string]any, dotted string) (any, bool) {
	current := any(root)
	for _, key := range strings.Split(dotted, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[key]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func assertEntry(t *testing.T, got, want map[string]any) {
	t.Helper()
	gotJSON, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("entry = %s, want %s", gotJSON, wantJSON)
	}
}

func toStrings(values []any) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok {
			result = append(result, text)
		}
	}
	return result
}

func TestGrokWritesTOMLHeaders(t *testing.T) {
	clearEnv(t)
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeGlobal, Root: root}
	if _, _, err := All()["grok"].Install(target, httpOptions()); err != nil {
		t.Fatalf("Install: %v", err)
	}
	doc := readTOML(t, filepath.Join(root, globalPathFor(t, "grok")))
	servers, ok := doc["mcp_servers"].(map[string]any)
	if !ok {
		t.Fatalf("mcp_servers = %#v", doc["mcp_servers"])
	}
	entry, ok := servers["demo"].(map[string]any)
	if !ok {
		t.Fatalf("demo entry = %#v", servers["demo"])
	}
	if entry["url"] != "https://example.test/mcp" || entry["enabled"] != true {
		t.Fatalf("entry = %#v", entry)
	}
	headers, ok := entry["headers"].(map[string]any)
	if !ok || headers["Authorization"] != "Bearer token" {
		t.Fatalf("headers = %#v", entry["headers"])
	}
}
