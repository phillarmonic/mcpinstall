package install

import (
	"os"
	"path/filepath"
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

func TestApplyGuidanceGlobalKeepsOtherHarnessLayout(t *testing.T) {
	root := t.TempDir()

	results, err := ApplyGuidance(testSpec, GuidanceRequest{Root: root, Harness: "copilot", Global: true})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, ".copilot", "instructions.md")
	if results["copilot"] != want {
		t.Fatalf("path = %q, want %q", results["copilot"], want)
	}
}

func TestApplyGuidanceSkipsExistingUnlessForced(t *testing.T) {
	root := t.TempDir()
	req := GuidanceRequest{Root: root, Harness: "copilot"}
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
