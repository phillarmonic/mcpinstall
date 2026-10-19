package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/phillarmonic/mcpinstall/platform"
)

func clearHookEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"XDG_CONFIG_HOME", "APPDATA",
		"CLAUDE_CONFIG_DIR", "CODEX_HOME", "GROK_HOME", "KIMI_CODE_HOME",
		"KIRO_HOME", "COPILOT_HOME",
	} {
		t.Setenv(name, "")
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
		t.Fatalf("parse %s: %v\n%s", path, err, data)
	}
	return value
}

func asMap(t *testing.T, value any) map[string]any {
	t.Helper()
	out, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("value is %T, want object", value)
	}
	return out
}

func asList(t *testing.T, value any) []any {
	t.Helper()
	out, ok := value.([]any)
	if !ok {
		t.Fatalf("value is %T, want array", value)
	}
	return out
}

func hook(event Event, command string) Hook {
	return Hook{Event: event, Command: command}
}

func TestNestedClaudeInstallAndRemovePreservesOtherHooks(t *testing.T) {
	clearHookEnv(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o750); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".claude", "settings.local.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	seed := map[string]any{
		"hooks": map[string]any{
			"SessionStart": []any{
				map[string]any{
					"matcher": "",
					"hooks":   []any{map[string]any{"type": "command", "command": "other-tool start"}},
				},
			},
		},
	}
	data, _ := json.Marshal(seed)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	target := platform.Target{Scope: platform.ScopeProject, Root: root}
	opts := Options{Marker: "backstage"}
	list := []Hook{
		hook(SessionStart, "/opt/backstage hook --event session-start"),
		{Event: PreToolUse, Matcher: "Bash", Command: "/opt/backstage hook --event pre-tool-use"},
	}
	paths, _, err := InstallOne(target, "claude", list, opts)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if len(paths) != 1 || paths[0] != path {
		t.Fatalf("paths = %v, want %s", paths, path)
	}

	cfg := readJSON(t, path)
	hooks := asMap(t, cfg["hooks"])
	start := asList(t, hooks["SessionStart"])
	if len(start) != 2 {
		t.Fatalf("SessionStart has %d entries, want the preserved entry plus ours", len(start))
	}
	if got := asMap(t, start[0])["hooks"].([]any)[0].(map[string]any)["command"]; got != "other-tool start" {
		t.Fatalf("unrelated hook changed: %v", got)
	}
	ours := asMap(t, start[1])
	if got := ours["hooks"].([]any)[0].(map[string]any)["command"]; got != "/opt/backstage hook --event session-start" {
		t.Fatalf("command = %v", got)
	}
	pre := asList(t, hooks["PreToolUse"])
	if got := asMap(t, pre[0])["matcher"]; got != "Bash" {
		t.Fatalf("matcher = %v", got)
	}

	again, _, err := InstallOne(target, "claude", list, opts)
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("second install changed %v", again)
	}

	removed, _, err := RemoveOne(target, "claude", list, opts)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if len(removed) != 1 {
		t.Fatalf("removed = %v", removed)
	}
	cfg = readJSON(t, path)
	hooks = asMap(t, cfg["hooks"])
	if start := asList(t, hooks["SessionStart"]); len(start) != 1 {
		t.Fatalf("unrelated SessionStart hook was not preserved: %v", start)
	}
	if _, exists := hooks["PreToolUse"]; exists {
		t.Fatalf("managed PreToolUse entry survived removal: %v", hooks["PreToolUse"])
	}
}

func TestInstallClearsManagedEntriesFromOtherEvents(t *testing.T) {
	clearHookEnv(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o750); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, ".claude", "settings.local.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	seed := map[string]any{
		"hooks": map[string]any{
			"OldEvent": []any{
				map[string]any{"matcher": "", "hooks": []any{map[string]any{"type": "command", "command": "backstage old"}}},
			},
		},
	}
	data, _ := json.Marshal(seed)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	target := platform.Target{Scope: platform.ScopeProject, Root: root}
	if _, _, err := InstallOne(target, "claude", []Hook{hook(SessionStart, "/opt/backstage start")}, Options{Marker: "backstage"}); err != nil {
		t.Fatalf("install: %v", err)
	}
	cfg := readJSON(t, path)
	hooks := asMap(t, cfg["hooks"])
	if _, exists := hooks["OldEvent"]; exists {
		t.Fatalf("stale managed event survived: %v", hooks["OldEvent"])
	}
	if _, exists := hooks["SessionStart"]; !exists {
		t.Fatal("new hook was not written")
	}
}

func TestDroidRefusesToShadowSettingsHooks(t *testing.T) {
	clearHookEnv(t)
	root := t.TempDir()
	settings := filepath.Join(root, ".factory", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settings), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings, []byte(`{"hooks":{"SessionStart":[]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	target := platform.Target{Scope: platform.ScopeGlobal, Root: root}
	list := []Hook{hook(SessionStart, "/opt/backstage start")}
	if _, _, err := InstallOne(target, "droid", list, Options{Marker: "backstage"}); err == nil || !strings.Contains(err.Error(), "keeps Droid hooks") {
		t.Fatalf("err = %v", err)
	}

	hooksPath := filepath.Join(root, ".factory", "hooks.json")
	if err := os.WriteFile(hooksPath, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := InstallOne(target, "droid", list, Options{Marker: "backstage"}); err != nil {
		t.Fatalf("install with hooks.json present: %v", err)
	}
}

func TestUnknownHookEventRejected(t *testing.T) {
	err := Validate([]Hook{{Event: Event("not-an-event"), Command: "backstage hook"}}, Options{Marker: "backstage"})
	if err == nil || !strings.Contains(err.Error(), "unsupported hook event") {
		t.Fatalf("err = %v", err)
	}
}

func TestKiroRejectsDifferentVersion(t *testing.T) {
	clearHookEnv(t)
	root := t.TempDir()
	home := filepath.Join(root, "kiro", "hooks")
	if err := os.MkdirAll(home, 0o750); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, "backstage.json")
	if err := os.WriteFile(path, []byte(`{"version":"v2","hooks":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KIRO_HOME", filepath.Join(root, "kiro"))
	target := platform.Target{Scope: platform.ScopeGlobal, Root: root}
	paths, warnings, err := InstallOne(target, "kiro", []Hook{hook(SessionStart, "/opt/backstage start")}, Options{Marker: "backstage"})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if len(paths) != 0 {
		t.Fatalf("paths = %v", paths)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "not v1") {
		t.Fatalf("warnings = %v", warnings)
	}
}

func TestInstallRejectsCommandWithoutMarker(t *testing.T) {
	clearHookEnv(t)
	target := platform.Target{Scope: platform.ScopeGlobal, Root: t.TempDir()}
	_, _, err := InstallOne(target, "claude", []Hook{hook(SessionStart, "/usr/bin/other start")}, Options{Marker: "backstage"})
	if err == nil || !strings.Contains(err.Error(), "must contain the marker") {
		t.Fatalf("err = %v", err)
	}
}

func TestMarkerValidation(t *testing.T) {
	if err := ValidateMarker("backstage"); err != nil {
		t.Fatalf("valid marker rejected: %v", err)
	}
	for _, marker := range []string{"", "bad marker", "../x", ".hidden"} {
		if err := ValidateMarker(marker); err == nil {
			t.Fatalf("marker %q accepted", marker)
		}
	}
}

func TestCursorFlatShapeAndRemoval(t *testing.T) {
	clearHookEnv(t)
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeGlobal, Root: root}
	opts := Options{Marker: "backstage"}
	list := []Hook{
		hook(SessionStart, "/opt/backstage hook --event session-start"),
		{Event: PreToolUse, Matcher: "Shell", Command: "/opt/backstage hook --event pre-tool-use"},
	}

	paths, _, err := InstallOne(target, "cursor", list, opts)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	path := filepath.Join(root, ".cursor", "hooks.json")
	if len(paths) != 1 || paths[0] != path {
		t.Fatalf("paths = %v, want %s", paths, path)
	}
	cfg := readJSON(t, path)
	if got := cfg["version"]; got != float64(1) {
		t.Fatalf("version = %v", got)
	}
	hooks := asMap(t, cfg["hooks"])
	start := asList(t, hooks["sessionStart"])
	entry := asMap(t, start[0])
	if entry["type"] != "command" || entry["command"] != "/opt/backstage hook --event session-start" {
		t.Fatalf("entry = %v", entry)
	}
	pre := asMap(t, asList(t, hooks["preToolUse"])[0])
	if pre["matcher"] != "Shell" {
		t.Fatalf("matcher = %v", pre["matcher"])
	}

	if _, _, err := RemoveOne(target, "cursor", list, opts); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("dedicated hooks file survived removal: %v", err)
	}
}

func TestGeminiMapsItsOwnEventVocabulary(t *testing.T) {
	clearHookEnv(t)
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeGlobal, Root: root}
	opts := Options{Marker: "backstage"}
	list := []Hook{
		hook(SessionStart, "/opt/backstage hook --event session-start"),
		hook(PreToolUse, "/opt/backstage hook --event pre-tool-use"),
		hook(PostToolUse, "/opt/backstage hook --event post-tool-use"),
		hook(PreCompact, "/opt/backstage hook --event pre-compact"),
		hook(UserPromptSubmit, "/opt/backstage hook --event user-prompt"),
	}

	paths, warnings, err := InstallOne(target, "gemini", list, opts)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("paths = %v", paths)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "no user-prompt event") {
		t.Fatalf("warnings = %v", warnings)
	}
	cfg := readJSON(t, filepath.Join(root, ".gemini", "settings.json"))
	hooks := asMap(t, cfg["hooks"])
	for _, name := range []string{"SessionStart", "BeforeTool", "AfterTool", "PreCompress"} {
		if _, ok := hooks[name]; !ok {
			t.Fatalf("%s missing from %v", name, hooks)
		}
	}
	if _, ok := hooks["UserPromptSubmit"]; ok {
		t.Fatal("unsupported event was written")
	}
}

func TestAntigravityUsesNamedGroupAndMixedShapes(t *testing.T) {
	clearHookEnv(t)
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeGlobal, Root: root}
	opts := Options{Marker: "backstage"}
	list := []Hook{
		hook(SessionStart, "/opt/backstage hook --event session-start"),
		hook(PreToolUse, "/opt/backstage hook --event pre-tool-use"),
	}

	if _, _, err := InstallOne(target, "antigravity", list, opts); err != nil {
		t.Fatalf("install: %v", err)
	}
	cfg := readJSON(t, filepath.Join(root, ".gemini", "config", "hooks.json"))
	group := asMap(t, cfg["backstage"])
	lifecycle := asMap(t, asList(t, group["PreInvocation"])[0])
	if _, ok := lifecycle["command"]; !ok {
		t.Fatalf("lifecycle entry is not flat: %v", lifecycle)
	}
	tool := asMap(t, asList(t, group["PreToolUse"])[0])
	if _, ok := tool["hooks"]; !ok {
		t.Fatalf("tool entry is not nested: %v", tool)
	}
}

func TestZeroUsesExecFormAndGeneratedIDs(t *testing.T) {
	clearHookEnv(t)
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeGlobal, Root: root}
	opts := Options{Marker: "backstage"}
	list := []Hook{{
		Event:   SessionStart,
		Command: "/opt/backstage",
		Args:    []string{"hook", "--event", "session-start"},
	}}

	paths, _, err := InstallOne(target, "zero", list, opts)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	path := filepath.Join(root, ".config", "zero", "hooks.json")
	if len(paths) != 1 || paths[0] != path {
		t.Fatalf("paths = %v", paths)
	}
	cfg := readJSON(t, path)
	if cfg["enabled"] != true {
		t.Fatalf("enabled = %v", cfg["enabled"])
	}
	entry := asMap(t, asList(t, cfg["hooks"])[0])
	if entry["id"] != "backstage-session-start" || entry["event"] != "sessionStart" {
		t.Fatalf("entry = %v", entry)
	}
	if entry["command"] != "/opt/backstage" {
		t.Fatalf("command = %v", entry["command"])
	}
	if args := asList(t, entry["args"]); len(args) != 3 || args[0] != "hook" {
		t.Fatalf("args = %v", entry["args"])
	}

	if _, _, err := RemoveOne(target, "zero", list, opts); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("dedicated hooks file survived removal: %v", err)
	}
}

func TestZCodeUsesProcessEntries(t *testing.T) {
	clearHookEnv(t)
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeGlobal, Root: root}
	opts := Options{Marker: "backstage"}
	list := []Hook{{
		Event:   UserPromptSubmit,
		Command: "/opt/backstage",
		Args:    []string{"hook", "--event", "user-prompt"},
	}}

	paths, _, err := InstallOne(target, "zcode", list, opts)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	path := filepath.Join(root, ".zcode", "cli", "config.json")
	if len(paths) != 1 || paths[0] != path {
		t.Fatalf("paths = %v", paths)
	}
	cfg := readJSON(t, path)
	hooksBlock := asMap(t, cfg["hooks"])
	if hooksBlock["enabled"] != true || hooksBlock["maxOutputBytes"] != float64(262144) {
		t.Fatalf("hooks block = %v", hooksBlock)
	}
	events := asMap(t, hooksBlock["events"])
	group := asMap(t, asList(t, events["UserPromptSubmit"])[0])
	handler := asMap(t, asList(t, group["hooks"])[0])
	if handler["type"] != "process" || handler["command"] != "/opt/backstage" {
		t.Fatalf("handler = %v", handler)
	}
	if handler["statusMessage"] != "backstage user-prompt" {
		t.Fatalf("statusMessage = %v", handler["statusMessage"])
	}

	if _, _, err := RemoveOne(target, "zcode", list, opts); err != nil {
		t.Fatalf("remove: %v", err)
	}
	cfg = readJSON(t, path)
	events = asMap(t, asMap(t, cfg["hooks"])["events"])
	if _, exists := events["UserPromptSubmit"]; exists {
		t.Fatalf("managed event survived removal: %v", events)
	}
}

func TestKimiWritesTOMLHooksAndPreservesConfig(t *testing.T) {
	clearHookEnv(t)
	root := t.TempDir()
	home := filepath.Join(root, "kimi")
	if err := os.MkdirAll(home, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte("model = \"kimi\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KIMI_CODE_HOME", home)
	target := platform.Target{Scope: platform.ScopeGlobal, Root: root}
	opts := Options{Marker: "backstage"}
	list := []Hook{
		hook(SessionStart, "/opt/backstage hook --event session-start"),
		hook(PostToolUseFailure, "/opt/backstage hook --event post-tool-use"),
	}

	paths, _, err := InstallOne(target, "kimi", list, opts)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	path := filepath.Join(home, "config.toml")
	if len(paths) != 1 || paths[0] != path {
		t.Fatalf("paths = %v", paths)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := toml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse TOML: %v\n%s", err, data)
	}
	if cfg["model"] != "kimi" {
		t.Fatalf("model key lost: %v", cfg)
	}
	entries, ok := cfg["hooks"].([]any)
	if !ok || len(entries) != 2 {
		t.Fatalf("hooks = %T %v", cfg["hooks"], cfg["hooks"])
	}
	first := entries[0].(map[string]any)
	if first["event"] != "SessionStart" || first["command"] != "/opt/backstage hook --event session-start" {
		t.Fatalf("first hook = %v", first)
	}

	if _, _, err := RemoveOne(target, "kimi", list, opts); err != nil {
		t.Fatalf("remove: %v", err)
	}
	data, _ = os.ReadFile(path)
	cfg = nil
	if err := toml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse TOML after remove: %v\n%s", err, data)
	}
	if cfg["model"] != "kimi" {
		t.Fatalf("model key lost on remove: %v", cfg)
	}
	if _, exists := cfg["hooks"]; exists {
		t.Fatalf("hooks remained after remove: %v", cfg["hooks"])
	}
}

func TestKiroV3WritesStandaloneRegistration(t *testing.T) {
	clearHookEnv(t)
	root := t.TempDir()
	t.Setenv("KIRO_HOME", filepath.Join(root, "kiro"))
	target := platform.Target{Scope: platform.ScopeGlobal, Root: root}
	opts := Options{Marker: "backstage"}
	list := []Hook{
		hook(SessionStart, "/opt/backstage hook --event session-start"),
		hook(PreToolUse, "/opt/backstage hook --event pre-tool-use"),
	}

	paths, _, err := InstallOne(target, "kiro", list, opts)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	path := filepath.Join(root, "kiro", "hooks", "backstage.json")
	if len(paths) != 1 || paths[0] != path {
		t.Fatalf("paths = %v", paths)
	}
	cfg := readJSON(t, path)
	if cfg["version"] != "v1" {
		t.Fatalf("version = %v", cfg["version"])
	}
	entries := asList(t, cfg["hooks"])
	if len(entries) != 2 {
		t.Fatalf("hooks = %v", entries)
	}
	first := asMap(t, entries[0])
	if first["name"] != "backstage-session-start" || first["trigger"] != "SessionStart" {
		t.Fatalf("entry = %v", first)
	}
	if first["timeout"] != float64(5) {
		t.Fatalf("SessionStart timeout = %v", first["timeout"])
	}
	action := asMap(t, first["action"])
	if action["type"] != "command" || action["command"] != "/opt/backstage hook --event session-start" {
		t.Fatalf("action = %v", action)
	}

	if _, _, err := RemoveOne(target, "kiro", list, opts); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("dedicated registration survived removal: %v", err)
	}
}

func TestJSONHookShapes(t *testing.T) {
	clearHookEnv(t)
	opts := Options{Marker: "backstage"}
	list := []Hook{hook(SessionStart, "/opt/backstage start")}

	cases := []struct {
		agent string
		path  func(string) string
		check func(*testing.T, map[string]any)
	}{
		{
			agent: "codex",
			path:  func(root string) string { return filepath.Join(root, ".codex", "hooks.json") },
			check: func(t *testing.T, cfg map[string]any) {
				entry := asMap(t, asList(t, asMap(t, cfg["hooks"])["SessionStart"])[0])
				if asMap(t, asList(t, entry["hooks"])[0])["command"] != "/opt/backstage start" {
					t.Fatalf("entry = %v", entry)
				}
			},
		},
		{
			agent: "grok",
			path:  func(root string) string { return filepath.Join(root, ".grok", "hooks", "backstage.json") },
			check: func(t *testing.T, cfg map[string]any) {
				if _, ok := asMap(t, cfg["hooks"])["SessionStart"]; !ok {
					t.Fatalf("cfg = %v", cfg)
				}
			},
		},
		{
			agent: "devin",
			path:  func(root string) string { return filepath.Join(root, ".devin", "hooks.v1.json") },
			check: func(t *testing.T, cfg map[string]any) {
				if _, ok := cfg["SessionStart"]; !ok {
					t.Fatalf("Devin did not write a root event map: %v", cfg)
				}
				if _, ok := cfg["hooks"]; ok {
					t.Fatalf("Devin gained a hooks wrapper: %v", cfg)
				}
			},
		},
		{
			agent: "droid",
			path:  func(root string) string { return filepath.Join(root, ".factory", "hooks.json") },
			check: func(t *testing.T, cfg map[string]any) {
				if _, ok := cfg["SessionStart"]; !ok {
					t.Fatalf("Droid did not write a root event map: %v", cfg)
				}
			},
		},
		{
			agent: "command-code",
			path:  func(root string) string { return filepath.Join(root, ".commandcode", "settings.json") },
			check: func(t *testing.T, cfg map[string]any) {
				entry := asMap(t, asList(t, asMap(t, cfg["hooks"])["SessionStart"])[0])
				if _, ok := entry["matcher"]; ok {
					t.Fatalf("Command Code entry has a matcher: %v", entry)
				}
				if _, ok := entry["hooks"]; !ok {
					t.Fatalf("Command Code entry is not nested: %v", entry)
				}
			},
		},
		{
			agent: "copilot-cli",
			path:  func(root string) string { return filepath.Join(root, ".copilot", "hooks", "backstage.json") },
			check: func(t *testing.T, cfg map[string]any) {
				if cfg["version"] != float64(1) {
					t.Fatalf("version = %v", cfg["version"])
				}
				entry := asMap(t, asList(t, asMap(t, cfg["hooks"])["SessionStart"])[0])
				if entry["command"] != "/opt/backstage start" {
					t.Fatalf("entry = %v", entry)
				}
				if _, ok := entry["matcher"]; ok {
					t.Fatalf("Copilot entry has a matcher: %v", entry)
				}
				if _, ok := entry["hooks"]; ok {
					t.Fatalf("Copilot entry is nested: %v", entry)
				}
			},
		},
		{
			agent: "muse",
			path:  func(root string) string { return filepath.Join(root, ".config", "muse", "settings.json") },
			check: func(t *testing.T, cfg map[string]any) {
				if cfg["schema_version"] != float64(1) {
					t.Fatalf("schema_version = %v", cfg["schema_version"])
				}
				if _, ok := asMap(t, cfg["hooks"])["SessionStart"]; !ok {
					t.Fatalf("cfg = %v", cfg)
				}
			},
		},
		{
			agent: "codebuddy",
			path:  func(root string) string { return filepath.Join(root, ".codebuddy", "settings.json") },
			check: func(t *testing.T, cfg map[string]any) {
				if _, ok := asMap(t, cfg["hooks"])["SessionStart"]; !ok {
					t.Fatalf("cfg = %v", cfg)
				}
			},
		},
		{
			agent: "ibm-bob",
			path:  func(root string) string { return filepath.Join(root, ".bob", "settings", "settings.json") },
			check: func(t *testing.T, cfg map[string]any) {
				if _, ok := asMap(t, cfg["hooks"])["SessionStart"]; !ok {
					t.Fatalf("cfg = %v", cfg)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.agent, func(t *testing.T) {
			root := t.TempDir()
			target := platform.Target{Scope: platform.ScopeGlobal, Root: root}
			paths, _, err := InstallOne(target, tc.agent, list, opts)
			if err != nil {
				t.Fatalf("install: %v", err)
			}
			if len(paths) != 1 || paths[0] != tc.path(root) {
				t.Fatalf("paths = %v, want %s", paths, tc.path(root))
			}
			tc.check(t, readJSON(t, paths[0]))
		})
	}
}

func TestDevinMapsPostCompact(t *testing.T) {
	clearHookEnv(t)
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeGlobal, Root: root}
	opts := Options{Marker: "backstage"}
	list := []Hook{hook(PostCompact, "/opt/backstage compact")}

	paths, _, err := InstallOne(target, "devin", list, opts)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	cfg := readJSON(t, paths[0])
	if _, ok := cfg["PostCompaction"]; !ok {
		t.Fatalf("PostCompaction missing: %v", cfg)
	}

	_, warnings, err := InstallOne(target, "devin", []Hook{hook(PreCompact, "/opt/backstage pre-compact")}, opts)
	if err != nil {
		t.Fatalf("pre-compact install: %v", err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "no pre-compact event") {
		t.Fatalf("warnings = %v", warnings)
	}
}

func TestResolveListExpandsAllAndRejectsUnsupported(t *testing.T) {
	all, err := ResolveList("all")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != len(Names()) {
		t.Fatalf("all = %v, want %d hosts", all, len(Names()))
	}
	if _, err := ResolveList("opencode"); err == nil || !strings.Contains(err.Error(), "TypeScript") {
		t.Fatalf("opencode err = %v", err)
	}
	alias, err := ResolveList("kiro-cli")
	if err != nil {
		t.Fatal(err)
	}
	if len(alias) != 1 || alias[0] != "kiro" {
		t.Fatalf("alias = %v", alias)
	}
}
