package install

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestApplyGuidanceWritesProjectLayout(t *testing.T) {
	root := t.TempDir()

	results, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "cursor"})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, ".cursor", "instructions.md")
	if results["cursor"] != want {
		t.Fatalf("path = %q, want %q", results["cursor"], want)
	}
	body, err := os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "Always Use Test Tools First") {
		t.Fatalf("project guidance missing body: %s", body)
	}
	if strings.Contains(string(body), "alwaysApply:") {
		t.Fatal("project cursor guidance should not wrap Cursor rule frontmatter")
	}
}

func TestApplyGuidanceGlobalCursorWritesAlwaysApplyRule(t *testing.T) {
	root := t.TempDir()

	results, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "cursor", Global: true})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, ".cursor", "rules", "test.mdc")
	if results["cursor"] != want {
		t.Fatalf("path = %q, want %q", results["cursor"], want)
	}
	body, err := os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.HasPrefix(text, "---\n") {
		t.Fatalf("missing frontmatter: %s", body)
	}
	if !strings.Contains(text, "alwaysApply: true") {
		t.Fatalf("missing alwaysApply: %s", body)
	}
	if !strings.Contains(text, "Always Use Test Tools First") {
		t.Fatalf("wrapped rule missing body: %s", body)
	}
}

func TestApplyGuidanceGlobalCopilotWritesPersonalInstructions(t *testing.T) {
	root := t.TempDir()

	results, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "copilot", Global: true})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, ".copilot", "copilot-instructions.md")
	if results["copilot"] != want {
		t.Fatalf("path = %q, want %q", results["copilot"], want)
	}
	body, err := os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), testBegin) || !strings.Contains(string(body), "Always Use Test Tools First") {
		t.Fatalf("copilot personal instructions missing managed body: %s", body)
	}

	removed, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "copilot", Global: true, Removing: true})
	if err != nil {
		t.Fatal(err)
	}
	if removed["copilot"] != want {
		t.Fatalf("removed = %q, want %q", removed["copilot"], want)
	}
	if _, statErr := os.Stat(want); !os.IsNotExist(statErr) {
		t.Fatal("copilot personal instructions survived uninstall")
	}
}

func TestApplyGuidanceSkipsExistingUnlessForced(t *testing.T) {
	root := t.TempDir()
	req := GuidanceRequest{Root: root, Harness: "cursor"}
	if _, err := ApplyGuidance(testSpec, req); err != nil {
		t.Fatal(err)
	}
	second, err := ApplyGuidance(testSpec, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 0 {
		t.Fatalf("repeat install reported changes: %v", second)
	}
	req.Force = true
	forced, err := ApplyGuidance(testSpec, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(forced) != 1 {
		t.Fatalf("forced install = %v", forced)
	}
}

func TestApplyGuidanceDryRunDoesNotWrite(t *testing.T) {
	root := t.TempDir()

	results, err := ApplyGuidance(testSpec, GuidanceRequest{
		Root:    root,
		Harness: "copilot",
		DryRun:  true,
		Global:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("dry run results = %v", results)
	}
	if _, err := os.Stat(results["copilot"]); !os.IsNotExist(err) {
		t.Fatal("dry run wrote a guidance file")
	}
}

func TestApplyGuidanceUninstallRemovesOnlyTheFile(t *testing.T) {
	root := t.TempDir()
	req := GuidanceRequest{Root: root, Harness: "cursor", Global: true}
	if _, err := ApplyGuidance(testSpec, req); err != nil {
		t.Fatal(err)
	}
	req.Removing = true
	removed, err := ApplyGuidance(testSpec, req)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, ".cursor", "rules", "test.mdc")
	if removed["cursor"] != want {
		t.Fatalf("removed = %q, want %q", removed["cursor"], want)
	}
	if _, statErr := os.Stat(want); !os.IsNotExist(statErr) {
		t.Fatal("global cursor rule survived uninstall")
	}
	if _, statErr := os.Stat(filepath.Join(root, ".cursor", "rules")); statErr != nil {
		t.Fatalf("rules directory should remain: %v", statErr)
	}

	empty, err := ApplyGuidance(testSpec, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("second uninstall reported changes: %v", empty)
	}
}

func TestApplyGuidanceCodexProjectWritesManagedAgentsFile(t *testing.T) {
	root := t.TempDir()
	user := []byte("# Project notes\nDo not drop this.\n")
	path := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(path, user, 0o600); err != nil {
		t.Fatal(err)
	}

	results, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "chatgpt"})
	if err != nil {
		t.Fatal(err)
	}
	if results["codex"] != path {
		t.Fatalf("path = %q, want %q", results["codex"], path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "# Project notes") || !strings.Contains(text, testBegin) {
		t.Fatalf("managed install lost surrounding text: %s", body)
	}

	second, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "openai"})
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 0 {
		t.Fatalf("repeat managed install reported changes: %v", second)
	}
}

func TestApplyGuidanceCodexGlobalWritesHomeAgentsFile(t *testing.T) {
	root := t.TempDir()

	results, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "codex", Global: true})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, ".codex", "AGENTS.md")
	if results["codex"] != want {
		t.Fatalf("path = %q, want %q", results["codex"], want)
	}

	req := GuidanceRequest{Root: root, Harness: "codex", Global: true, Removing: true}
	removed, err := ApplyGuidance(testSpec, req)
	if err != nil {
		t.Fatal(err)
	}
	if removed["codex"] != want {
		t.Fatalf("removed = %q", removed["codex"])
	}
	if _, statErr := os.Stat(want); !os.IsNotExist(statErr) {
		t.Fatal("global Codex AGENTS.md survived uninstall")
	}
}

func TestApplyGuidanceCodexUninstallKeepsUserAgentsText(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(path, []byte("keep\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "codex"}); err != nil {
		t.Fatal(err)
	}

	removed, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "codex", Removing: true})
	if err != nil {
		t.Fatal(err)
	}
	if removed["codex"] != path {
		t.Fatalf("removed = %q", removed["codex"])
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "keep\n" {
		t.Fatalf("leftover = %q", body)
	}
	if strings.Contains(string(body), testBegin) {
		t.Fatal("managed markers survived uninstall")
	}
}

func TestApplyGuidanceOpenCodeProjectWritesManagedAgentsFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "AGENTS.md")
	if err := os.WriteFile(path, []byte("# Project notes\nDo not drop this.\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	results, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "opencode"})
	if err != nil {
		t.Fatal(err)
	}
	if results["opencode"] != path {
		t.Fatalf("path = %q, want %q", results["opencode"], path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "# Project notes") || !strings.Contains(string(body), testBegin) {
		t.Fatalf("managed install lost surrounding text: %s", body)
	}

	second, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "opencode"})
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 0 {
		t.Fatalf("repeat managed install reported changes: %v", second)
	}
}

func TestApplyGuidanceOpenCodeGlobalWritesConfigAgentsFile(t *testing.T) {
	root := t.TempDir()

	results, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "opencode", Global: true})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, ".config", "opencode", "AGENTS.md")
	if results["opencode"] != want {
		t.Fatalf("path = %q, want %q", results["opencode"], want)
	}

	removed, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "opencode", Global: true, Removing: true})
	if err != nil {
		t.Fatal(err)
	}
	if removed["opencode"] != want {
		t.Fatalf("removed = %q", removed["opencode"])
	}
	if _, err := os.Stat(want); !os.IsNotExist(err) {
		t.Fatal("global OpenCode AGENTS.md survived uninstall")
	}
}

// OpenCode and Codex both read the project AGENTS.md, and their guidance bodies
// are identical, so installing both has to converge on one managed section.
func TestApplyGuidanceOpenCodeAndCodexShareOneAgentsSection(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "AGENTS.md")

	if _, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "codex"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "opencode"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(body), testBegin); got != 1 {
		t.Fatalf("AGENTS.md holds %d managed sections, want 1:\n%s", got, body)
	}

	// Removing one harness must not strip the section the other still relies on.
	if _, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "opencode", Removing: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("AGENTS.md still exists after removing the only managed section: %v", err)
	}
}

func TestApplyGuidanceClaudeWritesManagedClaudeMD(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".claude", "CLAUDE.md")
	user := "# Project notes\nKeep this.\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(user), 0o600); err != nil {
		t.Fatal(err)
	}

	results, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "claude"})
	if err != nil {
		t.Fatal(err)
	}
	if results["claude"] != path {
		t.Fatalf("path = %q, want %q", results["claude"], path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "# Project notes") || !strings.Contains(text, testBegin) {
		t.Fatalf("managed install lost user text: %s", body)
	}

	second, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "claude"})
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 0 {
		t.Fatalf("repeat managed install reported changes: %v", second)
	}

	if _, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "claude", Removing: true}); err != nil {
		t.Fatal(err)
	}
	body, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != user {
		t.Fatalf("leftover = %q, want %q", body, user)
	}
}

func TestApplyGuidanceClaudeGlobalWritesHomeClaudeMD(t *testing.T) {
	root := t.TempDir()

	results, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "claude", Global: true})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, ".claude", "CLAUDE.md")
	if results["claude"] != want {
		t.Fatalf("path = %q, want %q", results["claude"], want)
	}

	removed, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "claude", Global: true, Removing: true})
	if err != nil {
		t.Fatal(err)
	}
	if removed["claude"] != want {
		t.Fatalf("removed = %q, want %q", removed["claude"], want)
	}
	if _, statErr := os.Stat(want); !os.IsNotExist(statErr) {
		t.Fatal("global Claude CLAUDE.md survived uninstall")
	}
}

func TestApplyGuidanceCopilotWritesRepositoryInstructions(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".github", "copilot-instructions.md")
	user := "# Project notes\nKeep this.\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(user), 0o600); err != nil {
		t.Fatal(err)
	}

	results, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "copilot"})
	if err != nil {
		t.Fatal(err)
	}
	if results["copilot"] != path {
		t.Fatalf("path = %q, want %q", results["copilot"], path)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "# Project notes") || !strings.Contains(text, testBegin) {
		t.Fatalf("managed install lost user text: %s", body)
	}

	second, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "copilot"})
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 0 {
		t.Fatalf("repeat managed install reported changes: %v", second)
	}

	if _, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "copilot", Removing: true}); err != nil {
		t.Fatal(err)
	}
	body, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != user {
		t.Fatalf("leftover = %q, want %q", body, user)
	}
}

func TestCopilotJetBrainsConfigDirFor(t *testing.T) {
	env := func(values map[string]string) func(string) string {
		return func(name string) string { return values[name] }
	}
	for _, tc := range []struct {
		name string
		goos string
		env  map[string]string
		want string
	}{
		{name: "windows local app data", goos: "windows", env: map[string]string{"LOCALAPPDATA": "/local"}, want: "/local"},
		{name: "windows fallback", goos: "windows", want: filepath.Join("AppData", "Local")},
		{name: "posix xdg", goos: "linux", env: map[string]string{"XDG_CONFIG_HOME": "/xdg"}, want: "/xdg"},
		{name: "posix fallback", goos: "darwin", want: ".config"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := copilotJetBrainsConfigDirFor(tc.goos, env(tc.env)); got != tc.want {
				t.Fatalf("config dir = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestApplyGuidanceCopilotJetBrainsWritesGlobalInstructions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the default path is AppData/Local on Windows")
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	root := t.TempDir()

	results, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "copilot-jetbrains", Global: true})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, ".config", "github-copilot", "intellij", "global-copilot-instructions.md")
	if results["copilot-jetbrains"] != want {
		t.Fatalf("path = %q, want %q", results["copilot-jetbrains"], want)
	}
	body, err := os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), testBegin) || !strings.Contains(string(body), "Always Use Test Tools First") {
		t.Fatalf("jetbrains global instructions missing managed body: %s", body)
	}

	removed, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "copilot-jetbrains", Global: true, Removing: true})
	if err != nil {
		t.Fatal(err)
	}
	if removed["copilot-jetbrains"] != want {
		t.Fatalf("removed = %q, want %q", removed["copilot-jetbrains"], want)
	}
	if _, statErr := os.Stat(want); !os.IsNotExist(statErr) {
		t.Fatal("jetbrains global instructions survived uninstall")
	}
}

func TestApplyGuidanceCopilotJetBrainsHonorsXDGConfigHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("XDG_CONFIG_HOME is the POSIX branch")
	}
	root := t.TempDir()
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	results, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "copilot-jetbrains", Global: true})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(xdg, "github-copilot", "intellij", "global-copilot-instructions.md")
	if results["copilot-jetbrains"] != want {
		t.Fatalf("path = %q, want %q", results["copilot-jetbrains"], want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("XDG destination not written: %v", err)
	}
}

func TestApplyGuidanceCopilotAndJetBrainsShareRepositoryInstructions(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".github", "copilot-instructions.md")

	if _, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "copilot"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "copilot-jetbrains"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(body), testBegin); got != 1 {
		t.Fatalf(".github/copilot-instructions.md holds %d managed sections, want 1:\n%s", got, body)
	}

	if _, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "copilot-jetbrains", Removing: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("copilot-instructions.md still exists after removing the only managed section: %v", err)
	}
}
