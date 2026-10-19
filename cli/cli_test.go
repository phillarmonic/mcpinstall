package cli

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillarmonic/mcpinstall/platform/catalog"
)

// run invokes the CLI and returns its exit code with captured output.
func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(testSpec, args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// clearCLIEnv removes the relocation variables a developer machine may have
// set, so a test that writes global configuration stays inside its root.
func clearCLIEnv(t *testing.T) {
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

func TestNoArgumentsIsAUsageError(t *testing.T) {
	// Bare invocation is handled by main as the stdio server, so the CLI must
	// never treat an empty argument list as "serve".
	code, _, stderr := run(t)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "usage:") {
		t.Fatalf("stderr has no usage text: %q", stderr)
	}
}

func TestUnknownCommandIsAUsageError(t *testing.T) {
	code, _, stderr := run(t, "frobnicate")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestInstallRequiresHost(t *testing.T) {
	code, _, stderr := run(t, "install")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "--host") {
		t.Fatalf("stderr does not mention --host: %q", stderr)
	}
}

func TestInstallRejectsUnsupportedHost(t *testing.T) {
	code, _, stderr := run(t, "install", "--host", "vim", "--root", t.TempDir())
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "unsupported host") {
		t.Fatalf("stderr = %q", stderr)
	}
	if !strings.Contains(stderr, "all") {
		t.Fatalf("stderr does not advertise the all selector: %q", stderr)
	}
}

func TestInstallAllFillsInEveryProjectHost(t *testing.T) {
	root := t.TempDir()

	code, stdout, stderr := run(t, "install", "--host", "all", "--root", root)
	// User-level-only hosts are skipped with a warning, so one host that has
	// no project-scope file does not fail the whole run.
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout = %q\nstderr = %q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "copilot-jetbrains: warning: skipped: no project-level configuration") {
		t.Fatalf("stdout does not report the skip: %q", stdout)
	}
	for _, relative := range []string{
		".mcp.json",
		filepath.Join(".codex", "config.toml"),
		filepath.Join(".vscode", "mcp.json"),
		filepath.Join(".cursor", "mcp.json"),
	} {
		if _, err := os.Stat(filepath.Join(root, relative)); err != nil {
			t.Fatalf("expected %s: %v", relative, err)
		}
	}
}

func TestInstallAllGlobalScopeStaysInsideTheRoot(t *testing.T) {
	// These would otherwise point the user-level hosts at the developer's real
	// configuration, which is the failure this test exists to catch.
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("LOCALAPPDATA", "")
	t.Setenv("APPDATA", "")
	for _, name := range []string{
		"CLAUDE_CONFIG_DIR", "CODEX_HOME", "GROK_HOME", "KIMI_CODE_HOME",
		"KIRO_HOME", "COPILOT_HOME", "PRIME_AGENT_CODING_AGENT_DIR",
		"PI_CODING_AGENT_DIR", "OMP_PROFILE", "PI_PROFILE", "PI_CONFIG_DIR",
	} {
		t.Setenv(name, "")
	}

	root := t.TempDir()

	code, stdout, stderr := run(t, "install", "--host", "all", "--scope", "global", "--format", "json", "--root", root)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout = %q\nstderr = %q", code, stdout, stderr)
	}

	var written []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			written = append(written, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// A host writes at most one file, and a global-only run cannot write the
	// project-only host (swival) or a stdio entry for the HTTP-only host
	// (zcode), so the count is bounded by the catalog rather than equal to it.
	if len(written) == 0 || len(written) > len(catalog.Names()) {
		t.Fatalf("wrote %d files under the root, want at most one per host: %v", len(written), written)
	}
	for _, relative := range []string{
		filepath.Join(".gemini", "settings.json"),
		filepath.Join(".openclaw", "config.json"),
		filepath.Join(".grok", "config.toml"),
		filepath.Join(".config", "zed", "settings.json"),
	} {
		if _, err := os.Stat(filepath.Join(root, relative)); err != nil {
			t.Fatalf("expected %s: %v", relative, err)
		}
	}
}

func TestInstallHTTPFlagsWriteTheRemoteEntry(t *testing.T) {
	clearCLIEnv(t)
	root := t.TempDir()

	code, stdout, stderr := run(t,
		"install", "--host", "gemini", "--scope", "global", "--root", root,
		"--url", "https://example.test/mcp", "--header", "Authorization=Bearer token",
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout = %q\nstderr = %q", code, stdout, stderr)
	}

	data, err := os.ReadFile(filepath.Join(root, ".gemini", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	servers, ok := cfg["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf("mcpServers missing: %#v", cfg)
	}
	entry, ok := servers[testSpec.Name].(map[string]any)
	if !ok {
		t.Fatalf("entry missing: %#v", servers)
	}
	if entry["httpUrl"] != "https://example.test/mcp" {
		t.Fatalf("httpUrl = %v", entry["httpUrl"])
	}
	headers, ok := entry["headers"].(map[string]any)
	if !ok || headers["Authorization"] != "Bearer token" {
		t.Fatalf("headers = %#v", entry["headers"])
	}
}

func TestInstallHTTPTransportReachesTheHTTPOnlyHost(t *testing.T) {
	clearCLIEnv(t)
	root := t.TempDir()

	code, stdout, stderr := run(t,
		"install", "--host", "zcode", "--scope", "global", "--root", root,
		"--transport", "http", "--url", "https://example.test/mcp",
	)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout = %q\nstderr = %q", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(root, ".zcode", "cli", "config.json")); err != nil {
		t.Fatalf("expected the ZCode config: %v", err)
	}
}

func TestInstallRejectsUnsupportedScope(t *testing.T) {
	code, _, stderr := run(t, "install", "--host", "claude", "--scope", "galaxy", "--root", t.TempDir())
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "unsupported scope") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestInstallRejectsUnsupportedFormat(t *testing.T) {
	code, _, stderr := run(t, "install", "--host", "claude", "--format", "xml", "--root", t.TempDir())
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "unsupported format") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestInstallWritesRequestedHosts(t *testing.T) {
	root := t.TempDir()

	code, stdout, stderr := run(t, "install", "--host", "claude,cursor", "--root", root)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "claude: wrote .mcp.json") {
		t.Fatalf("stdout = %q", stdout)
	}
	if !strings.Contains(stdout, "cursor: wrote "+filepath.Join(".cursor", "mcp.json")) {
		t.Fatalf("stdout = %q", stdout)
	}
	for _, relative := range []string{".mcp.json", filepath.Join(".cursor", "mcp.json")} {
		if _, err := os.Stat(filepath.Join(root, relative)); err != nil {
			t.Fatalf("expected %s: %v", relative, err)
		}
	}
}

func TestInstallDryRunReportsWithoutWriting(t *testing.T) {
	root := t.TempDir()

	code, stdout, stderr := run(t, "install", "--host", "claude", "--dry-run", "--root", root)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "claude: would write .mcp.json") {
		t.Fatalf("stdout = %q", stdout)
	}
	if !strings.Contains(stdout, "1 would change") {
		t.Fatalf("summary does not describe a dry run: %q", stdout)
	}
	if _, err := os.Stat(filepath.Join(root, ".mcp.json")); !os.IsNotExist(err) {
		t.Fatal("dry run wrote a configuration file")
	}
}

func TestInstallSecondRunReportsAlreadyCurrent(t *testing.T) {
	root := t.TempDir()

	if code, _, stderr := run(t, "install", "--host", "claude", "--root", root); code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	code, stdout, stderr := run(t, "install", "--host", "claude", "--root", root)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "already current") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestUninstallWithNothingInstalledReportsNotInstalled(t *testing.T) {
	root := t.TempDir()

	code, stdout, stderr := run(t, "uninstall", "--host", "claude", "--root", root)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "claude: not installed") {
		t.Fatalf("stdout = %q", stdout)
	}
	if !strings.Contains(stdout, "0 removed, 1 not installed") {
		t.Fatalf("summary = %q", stdout)
	}
	// "already current" asserts the managed entry is present and up to date,
	// which is the opposite of what an uninstall found here.
	if strings.Contains(stdout, "already current") {
		t.Fatalf("uninstall reused install wording: %q", stdout)
	}
}

func TestUninstallRemovesConfiguration(t *testing.T) {
	root := t.TempDir()

	if code, _, stderr := run(t, "install", "--host", "claude", "--root", root); code != 0 {
		t.Fatalf("install exit code = %d, stderr = %q", code, stderr)
	}
	code, stdout, stderr := run(t, "uninstall", "--host", "claude", "--root", root)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "claude: removed .mcp.json") {
		t.Fatalf("stdout = %q", stdout)
	}
	if _, err := os.Stat(filepath.Join(root, ".mcp.json")); !os.IsNotExist(err) {
		t.Fatal("configuration file survived uninstall")
	}
}

func TestJSONFormatIsMachineReadable(t *testing.T) {
	root := t.TempDir()

	code, stdout, stderr := run(t, "install", "--host", "claude,codex", "--format", "json", "--root", root)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}

	payload := map[string]any{}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout)
	}
	if payload["action"] != "install" || payload["scope"] != "project" {
		t.Fatalf("unexpected envelope: %#v", payload)
	}
	results, ok := payload["results"].([]any)
	if !ok || len(results) != 2 {
		t.Fatalf("results = %#v", payload["results"])
	}
}

func TestCommandOverrideReachesEveryHost(t *testing.T) {
	root := t.TempDir()

	code, _, stderr := run(t, "install", "--host", "claude", "--command", "/opt/bin/kit", "--root", root)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}

	data, err := os.ReadFile(filepath.Join(root, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "/opt/bin/kit") {
		t.Fatalf("command override was not written: %s", data)
	}
}

func TestHostsListsEveryAdapter(t *testing.T) {
	code, stdout, stderr := run(t, "hosts")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	for _, name := range []string{
		"claude", "codex", "copilot", "copilot-jetbrains", "cursor", "opencode",
		"gemini", "openclaw", "grok", "muse", "prime-agent",
	} {
		if !strings.Contains(stdout, name) {
			t.Fatalf("stdout is missing %q: %q", name, stdout)
		}
	}
	for _, name := range []string{"dsh", "pi"} {
		found := false
		for line := range strings.SplitSeq(stdout, "\n") {
			if strings.HasPrefix(line, name) && strings.Contains(line, "guidance only") {
				found = true
			}
		}
		if !found {
			t.Fatalf("stdout does not mark %s as guidance only: %q", name, stdout)
		}
	}
	// The opencode row must report its own server key, not another host's.
	var opencodeRow string
	for line := range strings.SplitSeq(stdout, "\n") {
		if strings.HasPrefix(line, "opencode ") {
			opencodeRow = line
		}
	}
	if !strings.Contains(opencodeRow, "key=mcp") {
		t.Fatalf("opencode row does not report its server key: %q", stdout)
	}
	if !strings.Contains(stdout, "mcp_servers") {
		t.Fatalf("stdout does not report the codex server key: %q", stdout)
	}
	// The longest name must still be followed by its own columns, which is what
	// a fixed-width column would stop doing.
	if !strings.Contains(stdout, "copilot-jetbrains json") {
		t.Fatalf("the host column is misaligned: %q", stdout)
	}
	if !strings.Contains(stdout, "every supported MCP host listed above") {
		t.Fatalf("stdout does not explain the all selector: %q", stdout)
	}
}

func TestVersionAndHelp(t *testing.T) {
	if code, stdout, _ := run(t, "version"); code != 0 || !strings.Contains(stdout, Version) {
		t.Fatalf("version: code=%d stdout=%q", code, stdout)
	}
	if code, stdout, _ := run(t, "help"); code != 0 || !strings.Contains(stdout, "usage:") {
		t.Fatalf("help: code=%d stdout=%q", code, stdout)
	}
	if code, stdout, _ := run(t, "help"); code != 0 || !strings.Contains(stdout, "global-guidance") {
		t.Fatalf("help missing global-guidance: code=%d stdout=%q", code, stdout)
	}
}

func TestGlobalGuidanceRequiresSubcommand(t *testing.T) {
	code, _, stderr := run(t, "global-guidance")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "install|uninstall") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestGlobalGuidanceRejectsUnknownSubcommand(t *testing.T) {
	code, _, stderr := run(t, "global-guidance", "frobnicate")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "unknown command") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestGlobalGuidanceInstallsCursorRuleUnderRoot(t *testing.T) {
	root := t.TempDir()

	code, stdout, stderr := run(t, "global-guidance", "install", "--harness", "cursor", "--root", root)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	rel := filepath.Join(".cursor", "rules", "test.mdc")
	if !strings.Contains(stdout, "cursor: wrote "+rel) {
		t.Fatalf("stdout = %q", stdout)
	}
	body, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "alwaysApply: true") {
		t.Fatalf("missing alwaysApply: %s", body)
	}

	code, stdout, stderr = run(t, "global-guidance", "uninstall", "--harness", "cursor", "--root", root)
	if code != 0 {
		t.Fatalf("uninstall exit code = %d, stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "cursor: removed "+rel) {
		t.Fatalf("stdout = %q", stdout)
	}
	if _, err := os.Stat(filepath.Join(root, rel)); !os.IsNotExist(err) {
		t.Fatal("global cursor rule survived uninstall")
	}
}

func TestGlobalGuidanceDryRunReportsWithoutWriting(t *testing.T) {
	root := t.TempDir()

	code, stdout, stderr := run(t, "global-guidance", "install", "--harness", "copilot", "--dry-run", "--root", root)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "would write") {
		t.Fatalf("stdout = %q", stdout)
	}
	if _, err := os.Stat(filepath.Join(root, ".copilot", "instructions.md")); !os.IsNotExist(err) {
		t.Fatal("dry run wrote a guidance file")
	}
}

func TestInstallAcceptsChatGPTAsCodex(t *testing.T) {
	root := t.TempDir()

	code, stdout, stderr := run(t, "install", "--host", "chatgpt", "--root", root)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "codex: wrote "+filepath.Join(".codex", "config.toml")) {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestInstallGuidanceWritesCodexAgentsFile(t *testing.T) {
	root := t.TempDir()

	code, stdout, stderr := run(t, "install-guidance", "--harness", "codex", "--root", root)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "codex: wrote AGENTS.md") {
		t.Fatalf("stdout = %q", stdout)
	}
	body, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "test-guidance:begin") {
		t.Fatalf("missing managed markers: %s", body)
	}
}

func TestGlobalGuidanceInstallsCodexAgentsFile(t *testing.T) {
	root := t.TempDir()

	code, stdout, stderr := run(t, "global-guidance", "install", "--harness", "openai", "--root", root)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	rel := filepath.Join(".codex", "AGENTS.md")
	if !strings.Contains(stdout, "codex: wrote "+rel) {
		t.Fatalf("stdout = %q", stdout)
	}
}

// OpenCode names its server table "mcp" and takes the command as an array, and
// its user-level default filename differs from its project one. Both scopes are
// exercised here because the CLI is the only surface that resolves them.
func TestOpenCodeInstallAndUninstallRoundTrip(t *testing.T) {
	// OpenCode's user-level directory follows XDG_CONFIG_HOME, so clear it to
	// keep the run inside the temporary root.
	t.Setenv("XDG_CONFIG_HOME", "")
	root := t.TempDir()

	t.Run("project", func(t *testing.T) {
		path := filepath.Join(root, "opencode.json")
		if code, _, stderr := run(t, "install", "--host", "opencode", "--root", root); code != 0 {
			t.Fatalf("install: code=%d stderr=%q", code, stderr)
		}
		cfg := decodeObject(t, path)
		servers, ok := cfg["mcp"].(map[string]any)
		if !ok {
			t.Fatalf("no mcp table in %s: %#v", path, cfg)
		}
		entry, ok := servers["test-server"].(map[string]any)
		if !ok {
			t.Fatalf("no test-server entry: %#v", servers)
		}
		if entry["type"] != "local" {
			t.Errorf("type = %v, want local", entry["type"])
		}
		command, ok := entry["command"].([]any)
		if !ok || len(command) != 1 || command[0] != "test-command" {
			t.Errorf("command = %#v, want [test-command]", entry["command"])
		}

		if code, _, stderr := run(t, "uninstall", "--host", "opencode", "--root", root); code != 0 {
			t.Fatalf("uninstall: code=%d stderr=%q", code, stderr)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("emptied configuration survived uninstall: %v", err)
		}
	})

	t.Run("global", func(t *testing.T) {
		path := filepath.Join(root, ".config", "opencode", "opencode.jsonc")
		args := []string{"--host", "opencode", "--scope", "global", "--root", root}
		if code, _, stderr := run(t, append([]string{"install"}, args...)...); code != 0 {
			t.Fatalf("install: code=%d stderr=%q", code, stderr)
		}
		if servers, ok := decodeObject(t, path)["mcp"].(map[string]any); !ok || servers["test-server"] == nil {
			t.Fatalf("no aekf entry in %s", path)
		}

		if code, _, stderr := run(t, append([]string{"uninstall"}, args...)...); code != 0 {
			t.Fatalf("uninstall: code=%d stderr=%q", code, stderr)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("emptied configuration survived uninstall: %v", err)
		}
	})
}

func decodeObject(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := map[string]any{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("%s is not valid JSON: %v", path, err)
	}
	return cfg
}
