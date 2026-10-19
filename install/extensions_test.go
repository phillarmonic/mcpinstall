package install

import (
	"strings"
	"testing"

	"github.com/phillarmonic/mcpinstall"
	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/hooks"
	"github.com/phillarmonic/mcpinstall/platform/skills"
)

func clearExtensionEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"XDG_CONFIG_HOME", "APPDATA",
		"CLAUDE_CONFIG_DIR", "CODEX_HOME", "GROK_HOME", "KIMI_CODE_HOME",
		"KIRO_HOME", "COPILOT_HOME",
	} {
		t.Setenv(name, "")
	}
}

func extensionSpec() mcpinstall.Spec {
	spec := testSpec
	spec.Skills = []mcpinstall.Skill{{
		Name:    "demo",
		Content: "# Demo\n\nUse the demo skill.\n",
	}}
	spec.Hooks = []mcpinstall.Hook{
		{
			Event:   mcpinstall.HookSessionStart,
			Command: "test-command hook --event session-start",
		},
		{
			Event:   mcpinstall.HookPreToolUse,
			Command: "test-command hook --event pre-tool-use",
		},
	}
	return spec
}

func TestRunSkillsInstallsAndRemoves(t *testing.T) {
	root := t.TempDir()
	req := SkillRequest{
		Spec:   extensionSpec(),
		Agents: []string{"claude", "agents"},
		Scope:  platform.ScopeProject,
		Root:   root,
	}

	results, err := RunSkills(req, false)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}
	for _, result := range results {
		if len(result.Paths) != 1 {
			t.Fatalf("%s paths = %v", result.Agent, result.Paths)
		}
	}
	if results[0].Paths[0] != ".claude/skills/demo/SKILL.md" {
		t.Fatalf("claude path = %q", results[0].Paths[0])
	}
	if results[1].Paths[0] != ".agents/skills/demo/SKILL.md" {
		t.Fatalf("agents path = %q", results[1].Paths[0])
	}

	again, err := RunSkills(req, false)
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	for _, result := range again {
		if len(result.Paths) != 0 {
			t.Fatalf("%s changed on repeat: %v", result.Agent, result.Paths)
		}
	}

	removed, err := RunSkills(req, true)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	for _, result := range removed {
		if len(result.Paths) != 1 {
			t.Fatalf("%s removed paths = %v", result.Agent, result.Paths)
		}
	}
}

func TestRunHooksSkipsGlobalOnlyHostsAtProjectScope(t *testing.T) {
	root := t.TempDir()
	req := HookRequest{
		Spec:  extensionSpec(),
		Hosts: []string{"claude", "codex"},
		Scope: platform.ScopeProject,
		Root:  root,
	}

	results, err := RunHooks(req, false)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %d", len(results))
	}
	if results[0].Error != "" || len(results[0].Paths) != 1 {
		t.Fatalf("claude result = %+v", results[0])
	}
	if results[1].Error != "" || len(results[1].Paths) != 0 {
		t.Fatalf("codex result = %+v", results[1])
	}
	if len(results[1].Warnings) != 1 || !strings.Contains(results[1].Warnings[0], "no project-level hooks configuration") {
		t.Fatalf("codex warnings = %v", results[1].Warnings)
	}
}

func TestRunHooksInstallsAndRemoves(t *testing.T) {
	clearExtensionEnv(t)
	root := t.TempDir()
	req := HookRequest{
		Spec:  extensionSpec(),
		Hosts: []string{"claude", "cursor"},
		Scope: platform.ScopeGlobal,
		Root:  root,
	}

	results, err := RunHooks(req, false)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %d", len(results))
	}
	if results[0].Paths[0] != ".claude/settings.json" {
		t.Fatalf("claude path = %q", results[0].Paths[0])
	}
	if results[1].Paths[0] != ".cursor/hooks.json" {
		t.Fatalf("cursor path = %q", results[1].Paths[0])
	}

	again, err := RunHooks(req, false)
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	for _, result := range again {
		if len(result.Paths) != 0 {
			t.Fatalf("%s changed on repeat: %v", result.Host, result.Paths)
		}
	}

	removed, err := RunHooks(req, true)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	for _, result := range removed {
		if len(result.Paths) != 1 {
			t.Fatalf("%s removed paths = %v", result.Host, result.Paths)
		}
	}
}

func TestRunHooksInstallsEverySupportedHost(t *testing.T) {
	clearExtensionEnv(t)
	root := t.TempDir()
	req := HookRequest{
		Spec:  extensionSpec(),
		Hosts: hooks.Names(),
		Scope: platform.ScopeGlobal,
		Root:  root,
	}

	results, err := RunHooks(req, false)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if len(results) != len(hooks.Names()) {
		t.Fatalf("results = %d, want %d", len(results), len(hooks.Names()))
	}
	for _, result := range results {
		if result.Error != "" {
			t.Fatalf("%s: %s", result.Host, result.Error)
		}
		if len(result.Paths) != 1 {
			t.Fatalf("%s paths = %v", result.Host, result.Paths)
		}
	}

	removed, err := RunHooks(req, true)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	for _, result := range removed {
		if result.Error != "" {
			t.Fatalf("%s: %s", result.Host, result.Error)
		}
		if len(result.Paths) != 1 {
			t.Fatalf("%s removed paths = %v", result.Host, result.Paths)
		}
	}
}

func TestRunSkillsInstallsEveryHost(t *testing.T) {
	clearExtensionEnv(t)
	root := t.TempDir()
	req := SkillRequest{
		Spec:   extensionSpec(),
		Agents: skills.Names(),
		Scope:  platform.ScopeProject,
		Root:   root,
	}

	results, err := RunSkills(req, false)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if len(results) != len(skills.Names()) {
		t.Fatalf("results = %d, want %d", len(results), len(skills.Names()))
	}
	for _, result := range results {
		if result.Error != "" {
			t.Fatalf("%s: %s", result.Agent, result.Error)
		}
		if len(result.Paths) != 1 {
			t.Fatalf("%s paths = %v", result.Agent, result.Paths)
		}
	}

	removed, err := RunSkills(req, true)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	for _, result := range removed {
		if result.Error != "" {
			t.Fatalf("%s: %s", result.Agent, result.Error)
		}
		if len(result.Paths) != 1 {
			t.Fatalf("%s removed paths = %v", result.Agent, result.Paths)
		}
	}
}

func TestRunHooksRejectsUnmarkedCommand(t *testing.T) {
	clearExtensionEnv(t)
	req := HookRequest{
		Spec:  extensionSpec(),
		Hosts: []string{"claude"},
		Scope: platform.ScopeGlobal,
		Root:  t.TempDir(),
	}
	req.Spec.Hooks = []mcpinstall.Hook{{
		Event:   mcpinstall.HookSessionStart,
		Command: "another-tool start",
	}}

	_, err := RunHooks(req, false)
	if err == nil || !strings.Contains(err.Error(), "must contain the marker") {
		t.Fatalf("err = %v", err)
	}
}
