package copilotjetbrains

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

const settingsLabel = "Copilot for JetBrains MCP settings"

// pluginDir is where the plugin keeps its MCP configuration, relative to the
// platform's configuration directory.
//
// The "intellij" segment is a fixed literal rather than the running product:
// com/github/copilot/util/PathKt in the plugin's core jar resolves
// ".config/github-copilot/intellij" and "AppData/Local/github-copilot/intellij"
// as constant strings, so GoLand, PyCharm and IntelliJ IDEA all share the one
// file. Re-verify against the jar before changing it.
var pluginDir = filepath.Join("github-copilot", "intellij")

// errUserLevelOnly reports that the host has no project-scope configuration.
// The plugin reads a single file under the user's configuration directory and
// offers no per-project equivalent, so a project-scope run has nowhere to write.
var errUserLevelOnly = errors.New("no project-level configuration is available; pass --scope global")

// mcpPath returns the file the JetBrains plugin reads its MCP servers from.
// Only global scope has a path: the file is user-level wherever it lives, so a
// project-scope request is refused rather than quietly written outside the
// project root the caller asked to stay within.
func mcpPath(target platform.Target) (string, error) {
	if target.Scope != platform.ScopeGlobal {
		return "", errUserLevelOnly
	}
	return filepath.Join(configDir(target.Root), pluginDir, "mcp.json"), nil
}

// configDir returns the directory the platform keeps application configuration
// in, which the plugin resolves its own directory against.
func configDir(home string) string {
	return configDirFor(runtime.GOOS, home, os.Getenv)
}

// configDirFor is configDir with the platform and environment supplied, so both
// branches stay testable from whichever machine runs the suite.
func configDirFor(goos, home string, lookupEnv func(string) string) string {
	switch goos {
	case "windows":
		if local := lookupEnv("LOCALAPPDATA"); local != "" {
			return local
		}
		return filepath.Join(home, "AppData", "Local")
	default:
		if configHome := lookupEnv("XDG_CONFIG_HOME"); configHome != "" {
			return configHome
		}
		return filepath.Join(home, ".config")
	}
}

// upsertMCP merges the managed entry into the JetBrains configuration. The
// plugin names its server table "servers", the same key VS Code uses, and has
// never used the older "mcpServers" spelling. The strict writer is used so an
// unrelated "mcpServers" object is left as the user wrote it rather than being
// renamed onto a key that would make those servers live.
func upsertMCP(path string, opts common.Options) (bool, string, error) {
	return common.UpsertMCPServerInKeyOnly(
		path,
		settingsLabel,
		"servers",
		opts.ServerName,
		common.ManagedServer(opts),
		opts,
	)
}

func removeMCP(path string, opts common.Options) (bool, string, error) {
	return common.RemoveMCPServerInKeyOnly(path, settingsLabel, "servers", common.EntryName(opts), opts)
}
