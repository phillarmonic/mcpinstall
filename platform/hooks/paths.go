package hooks

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/phillarmonic/mcpinstall/platform"
)

var errGlobalOnly = errors.New("no project-level hooks configuration is available; pass --scope global")

func globalPath(target platform.Target, elem ...string) (string, error) {
	if target.Scope != platform.ScopeGlobal {
		return "", errGlobalOnly
	}
	return filepath.Join(append([]string{target.Root}, elem...)...), nil
}

func claudePath(target platform.Target, _ string) (string, error) {
	if target.Scope == platform.ScopeProject {
		return filepath.Join(repoRoot(target.Root), ".claude", "settings.local.json"), nil
	}
	return filepath.Join(envRoot("CLAUDE_CONFIG_DIR", target.Rel(".claude")), "settings.json"), nil
}

func codexPath(target platform.Target, _ string) (string, error) {
	if target.Scope != platform.ScopeGlobal {
		return "", errGlobalOnly
	}
	return filepath.Join(envRoot("CODEX_HOME", target.Rel(".codex")), "hooks.json"), nil
}

func cursorPath(target platform.Target, _ string) (string, error) {
	return globalPath(target, ".cursor", "hooks.json")
}

func geminiPath(target platform.Target, _ string) (string, error) {
	return globalPath(target, ".gemini", "settings.json")
}

func antigravityPath(target platform.Target, _ string) (string, error) {
	return globalPath(target, ".gemini", "config", "hooks.json")
}

func grokPath(target platform.Target, marker string) (string, error) {
	if target.Scope != platform.ScopeGlobal {
		return "", errGlobalOnly
	}
	return filepath.Join(envRoot("GROK_HOME", target.Rel(".grok")), "hooks", marker+".json"), nil
}

func devinPath(target platform.Target, _ string) (string, error) {
	return globalPath(target, ".devin", "hooks.v1.json")
}

func commandCodePath(target platform.Target, _ string) (string, error) {
	return globalPath(target, ".commandcode", "settings.json")
}

func copilotCLIPath(target platform.Target, marker string) (string, error) {
	if target.Scope != platform.ScopeGlobal {
		return "", errGlobalOnly
	}
	return filepath.Join(envRoot("COPILOT_HOME", target.Rel(".copilot")), "hooks", marker+".json"), nil
}

func droidPath(target platform.Target, _ string) (string, error) {
	return globalPath(target, ".factory", "hooks.json")
}

func musePath(target platform.Target, _ string) (string, error) {
	if target.Scope != platform.ScopeGlobal {
		return "", errGlobalOnly
	}
	return filepath.Join(configHome(target.Root), "muse", "settings.json"), nil
}

func codebuddyPath(target platform.Target, _ string) (string, error) {
	return globalPath(target, ".codebuddy", "settings.json")
}

func ibmBobPath(target platform.Target, _ string) (string, error) {
	return globalPath(target, ".bob", "settings", "settings.json")
}

func zeroPath(target platform.Target, _ string) (string, error) {
	if target.Scope != platform.ScopeGlobal {
		return "", errGlobalOnly
	}
	return filepath.Join(configHome(target.Root), "zero", "hooks.json"), nil
}

func zcodePath(target platform.Target, _ string) (string, error) {
	return globalPath(target, ".zcode", "cli", "config.json")
}

func kimiPath(target platform.Target, _ string) (string, error) {
	if target.Scope != platform.ScopeGlobal {
		return "", errGlobalOnly
	}
	return filepath.Join(envRoot("KIMI_CODE_HOME", target.Rel(".kimi-code")), "config.toml"), nil
}

func kiroPath(target platform.Target, marker string) (string, error) {
	if target.Scope != platform.ScopeGlobal {
		return "", errGlobalOnly
	}
	return filepath.Join(envRoot("KIRO_HOME", target.Rel(".kiro")), "hooks", marker+".json"), nil
}

func envRoot(variable, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(variable)); value != "" {
		return value
	}
	return fallback
}

func configHome(home string) string {
	if value := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(home, ".config")
}

// repoRoot finds the repository root Claude Code associates a project-level
// settings.local.json with: the nearest ancestor holding .git. It falls back to
// the invocation directory.
func repoRoot(start string) string {
	resolved, err := filepath.Abs(start)
	if err != nil {
		resolved = start
	}
	current := resolved
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return resolved
		}
		current = parent
	}
}
