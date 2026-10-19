package hosts

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/phillarmonic/mcpinstall/platform"
)

const (
	// geminiTimeoutMs is the timeout Gemini CLI and Antigravity document for
	// an MCP stdio server.
	geminiTimeoutMs = 5000
)

var (
	errGlobalOnly  = errors.New("no project-level configuration is available; pass --scope global")
	errProjectOnly = errors.New("no global-level configuration is available; pass --scope project")
)

// globalPath resolves a path under the target root for a global-only host. A
// project-scope request is refused rather than silently writing outside the
// project the caller asked to stay within.
func globalPath(target platform.Target, elem ...string) (string, error) {
	if target.Scope != platform.ScopeGlobal {
		return "", errGlobalOnly
	}
	return filepath.Join(append([]string{target.Root}, elem...)...), nil
}

// envPath resolves a path rooted at an environment override when it is set,
// otherwise under the home-derived default.
func envPath(override, fallback string, elem ...string) string {
	root := fallback
	if value := strings.TrimSpace(os.Getenv(override)); value != "" {
		root = value
	}
	return filepath.Join(append([]string{root}, elem...)...)
}

// configHome returns the XDG configuration root, falling back to ~/.config.
func configHome(home string) string {
	if value := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(home, ".config")
}

// appDataDir returns the roaming application-data root on Windows, falling
// back to the conventional path under home.
func appDataDir(home string) string {
	if value := strings.TrimSpace(os.Getenv("APPDATA")); value != "" {
		return value
	}
	return filepath.Join(home, "AppData", "Roaming")
}

func claudeDesktopPath(target platform.Target) (string, error) {
	if target.Scope != platform.ScopeGlobal {
		return "", errGlobalOnly
	}
	home := target.Root
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"), nil
	case "windows":
		return filepath.Join(appDataDir(home), "Claude", "claude_desktop_config.json"), nil
	default:
		return filepath.Join(configHome(home), "Claude", "claude_desktop_config.json"), nil
	}
}

func geminiPath(target platform.Target) (string, error) {
	return globalPath(target, ".gemini", "settings.json")
}

func openclawPath(target platform.Target) (string, error) {
	return globalPath(target, ".openclaw", "config.json")
}

func ompPath(target platform.Target) (string, error) {
	if target.Scope != platform.ScopeGlobal {
		return "", errGlobalOnly
	}
	// OMP's own precedence: a named profile (OMP_PROFILE, then PI_PROFILE)
	// owns <root>/profiles/<name>/agent and ignores PI_CODING_AGENT_DIR; the
	// default profile honors that variable, else <root>/agent. PI_CONFIG_DIR
	// renames the ~/.omp root.
	if profile := ompProfile(); profile != "" {
		return filepath.Join(ompConfigRoot(target.Root), "profiles", profile, "agent", "mcp.json"), nil
	}
	if dir := strings.TrimSpace(os.Getenv("PI_CODING_AGENT_DIR")); dir != "" {
		return filepath.Join(dir, "mcp.json"), nil
	}
	return filepath.Join(ompConfigRoot(target.Root), "agent", "mcp.json"), nil
}

// ompProfile returns the selected profile, treating empty and "default" as the
// default profile. OMP_PROFILE masks PI_PROFILE even when it is empty.
func ompProfile() string {
	if value, set := os.LookupEnv("OMP_PROFILE"); set {
		return normalizeOMPProfile(value)
	}
	return normalizeOMPProfile(os.Getenv("PI_PROFILE"))
}

func normalizeOMPProfile(value string) string {
	name := strings.TrimSpace(value)
	if name == "" || name == "default" {
		return ""
	}
	return name
}

// ompConfigRoot joins ~/.omp (or PI_CONFIG_DIR) under the home directory. A
// leading separator is dropped first, because OMP joins the value under the
// home rather than letting an absolute value replace it.
func ompConfigRoot(home string) string {
	name := strings.TrimSpace(os.Getenv("PI_CONFIG_DIR"))
	if name == "" {
		name = ".omp"
	}
	name = strings.TrimLeft(name, "/\\")
	return filepath.Join(home, name)
}

func antigravityPath(target platform.Target) (string, error) {
	return globalPath(target, ".gemini", "config", "mcp_config.json")
}

func zeroPath(target platform.Target) (string, error) {
	if target.Scope != platform.ScopeGlobal {
		return "", errGlobalOnly
	}
	return filepath.Join(configHome(target.Root), "zero", "config.json"), nil
}

func zcodePath(target platform.Target) (string, error) {
	return globalPath(target, ".zcode", "cli", "config.json")
}

func devinPath(target platform.Target) (string, error) {
	return globalPath(target, ".devin", "config.json")
}

func grokPath(target platform.Target) (string, error) {
	if target.Scope != platform.ScopeGlobal {
		return "", errGlobalOnly
	}
	return envPath("GROK_HOME", filepath.Join(target.Root, ".grok"), "config.toml"), nil
}

func kimiPath(target platform.Target) (string, error) {
	if target.Scope != platform.ScopeGlobal {
		return "", errGlobalOnly
	}
	return envPath("KIMI_CODE_HOME", filepath.Join(target.Root, ".kimi-code"), "mcp.json"), nil
}

func kiroPath(target platform.Target) (string, error) {
	if target.Scope != platform.ScopeGlobal {
		return "", errGlobalOnly
	}
	return envPath("KIRO_HOME", filepath.Join(target.Root, ".kiro"), "settings", "mcp.json"), nil
}

func commandCodePath(target platform.Target) (string, error) {
	return globalPath(target, ".commandcode", "mcp.json")
}

func ibmBobPath(target platform.Target) (string, error) {
	return globalPath(target, ".bob", "settings", "mcp.json")
}

func copilotCLIPath(target platform.Target) (string, error) {
	if target.Scope != platform.ScopeGlobal {
		return "", errGlobalOnly
	}
	return envPath("COPILOT_HOME", filepath.Join(target.Root, ".copilot"), "mcp-config.json"), nil
}

func droidPath(target platform.Target) (string, error) {
	return globalPath(target, ".factory", "mcp.json")
}

func primeAgentPath(target platform.Target) (string, error) {
	if target.Scope != platform.ScopeGlobal {
		return "", errGlobalOnly
	}
	return envPath("PRIME_AGENT_CODING_AGENT_DIR", filepath.Join(target.Root, ".prime", "agent"), "settings.json"), nil
}

func zedPath(target platform.Target) (string, error) {
	if target.Scope != platform.ScopeGlobal {
		return "", errGlobalOnly
	}
	home := target.Root
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(appDataDir(home), "Zed", "settings.json"), nil
	default:
		// Zed uses ~/.config/zed on macOS as well as Linux.
		return filepath.Join(configHome(home), "zed", "settings.json"), nil
	}
}

func musePath(target platform.Target) (string, error) {
	if target.Scope != platform.ScopeGlobal {
		return "", errGlobalOnly
	}
	return filepath.Join(configHome(target.Root), "muse", "settings.json"), nil
}

func swivalPath(target platform.Target) (string, error) {
	if target.Scope != platform.ScopeProject {
		return "", errProjectOnly
	}
	return filepath.Join(swivalRoot(target.Root), ".swival", "mcp.json"), nil
}

// swivalRoot mirrors Swival's own discovery: the nearest ancestor holding a
// .git directory or swival.toml, falling back to the invocation directory.
func swivalRoot(start string) string {
	resolved, err := filepath.Abs(start)
	if err != nil {
		resolved = start
	}
	current := resolved
	for {
		if _, err := os.Stat(filepath.Join(current, ".git")); err == nil {
			return current
		}
		if _, err := os.Stat(filepath.Join(current, "swival.toml")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return resolved
		}
		current = parent
	}
}

// opencode2Path follows OpenCode's own filename probe, the same one the V1
// adapter uses, because V2 reads the same configuration file.
func opencode2Path(target platform.Target) (string, error) {
	if target.Scope == platform.ScopeGlobal {
		return firstExisting(filepath.Join(configHome(target.Root), "opencode"), "opencode.jsonc", "opencode.json", "config.json"), nil
	}
	return firstExisting(target.Root,
		"opencode.json",
		"opencode.jsonc",
		filepath.Join(".opencode", "opencode.json"),
		filepath.Join(".opencode", "opencode.jsonc"),
	), nil
}

func firstExisting(dir string, candidates ...string) string {
	for _, name := range candidates {
		candidate := filepath.Join(dir, name)
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate
		}
	}
	return filepath.Join(dir, candidates[0])
}
