package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/phillarmonic/mcpinstall"
)

func extensionCLISpec() mcpinstall.Spec {
	spec := testSpec
	spec.Skills = []mcpinstall.Skill{{
		Name:    "demo",
		Content: "# Demo\n\nUse the demo skill.\n",
	}}
	spec.Hooks = []mcpinstall.Hook{{
		Event:   mcpinstall.HookSessionStart,
		Command: "test-command hook --event session-start",
	}}
	return spec
}

func runSpec(t *testing.T, spec mcpinstall.Spec, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(spec, args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestInstallSkillsRequiresAgent(t *testing.T) {
	code, _, stderr := runSpec(t, extensionCLISpec(), "install-skills")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "--agent") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestInstallSkillsWritesAndRemoves(t *testing.T) {
	clearCLIEnv(t)
	root := t.TempDir()
	spec := extensionCLISpec()

	code, stdout, stderr := runSpec(t, spec, "install-skills", "--agent", "claude,agents", "--scope", "project", "--root", root)
	if code != 0 {
		t.Fatalf("exit code = %d\nstdout = %q\nstderr = %q", code, stdout, stderr)
	}
	for _, want := range []string{".claude/skills/demo/SKILL.md", ".agents/skills/demo/SKILL.md"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout does not mention %s:\n%s", want, stdout)
		}
	}

	code, stdout, stderr = runSpec(t, spec, "uninstall-skills", "--agent", "claude,agents", "--scope", "project", "--root", root)
	if code != 0 {
		t.Fatalf("uninstall exit code = %d\nstdout = %q\nstderr = %q", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "removed") {
		t.Fatalf("stdout = %q", stdout)
	}
}

func TestInstallSkillsJSON(t *testing.T) {
	clearCLIEnv(t)
	root := t.TempDir()

	code, stdout, stderr := runSpec(t, extensionCLISpec(), "install-skills", "--agent", "claude", "--root", root, "--format", "json")
	if code != 0 {
		t.Fatalf("exit code = %d\nstdout = %q\nstderr = %q", code, stdout, stderr)
	}
	var payload struct {
		Action  string `json:"action"`
		Results []struct {
			Agent string   `json:"agent"`
			Paths []string `json:"paths"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("parse JSON: %v\n%s", err, stdout)
	}
	if payload.Action != "install-skills" || len(payload.Results) != 1 {
		t.Fatalf("payload = %+v", payload)
	}
	if payload.Results[0].Agent != "claude" || len(payload.Results[0].Paths) != 1 {
		t.Fatalf("result = %+v", payload.Results[0])
	}
}

func TestInstallHooksRequiresAgent(t *testing.T) {
	code, _, stderr := runSpec(t, extensionCLISpec(), "install-hooks")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "--agent") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestInstallHooksJSONReportsScopedSkip(t *testing.T) {
	clearCLIEnv(t)
	root := t.TempDir()

	code, stdout, stderr := runSpec(t, extensionCLISpec(), "install-hooks", "--agent", "claude,codex", "--scope", "project", "--root", root, "--format", "json")
	if code != 0 {
		t.Fatalf("exit code = %d\nstdout = %q\nstderr = %q", code, stdout, stderr)
	}
	var payload struct {
		Results []struct {
			Host     string   `json:"host"`
			Paths    []string `json:"paths"`
			Warnings []string `json:"warnings"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("parse JSON: %v\n%s", err, stdout)
	}
	if len(payload.Results) != 2 {
		t.Fatalf("results = %+v", payload.Results)
	}
	if payload.Results[0].Host != "claude" || len(payload.Results[0].Paths) != 1 {
		t.Fatalf("claude result = %+v", payload.Results[0])
	}
	if payload.Results[1].Host != "codex" || len(payload.Results[1].Warnings) != 1 {
		t.Fatalf("codex result = %+v", payload.Results[1])
	}
}

func TestInstallHooksGlobalWrites(t *testing.T) {
	clearCLIEnv(t)
	root := t.TempDir()

	code, stdout, stderr := runSpec(t, extensionCLISpec(), "install-hooks", "--agent", "claude,cursor", "--scope", "global", "--root", root)
	if code != 0 {
		t.Fatalf("exit code = %d\nstdout = %q\nstderr = %q", code, stdout, stderr)
	}
	for _, want := range []string{".claude/settings.json", ".cursor/hooks.json"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout does not mention %s:\n%s", want, stdout)
		}
	}
}

func TestInstallHooksRejectsUnsupportedHost(t *testing.T) {
	code, _, stderr := runSpec(t, extensionCLISpec(), "install-hooks", "--agent", "opencode", "--root", t.TempDir())
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr, "TypeScript") {
		t.Fatalf("stderr = %q", stderr)
	}
}
