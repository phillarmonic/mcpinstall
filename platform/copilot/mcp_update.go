package copilot

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

const settingsLabel = "Copilot VS Code MCP settings"

// mcpPath returns the VS Code file holding MCP servers for the target scope.
// Project scope writes the workspace file; user scope writes the profile file,
// which lives outside the home directory root on macOS and Windows.
func mcpPath(target platform.Target) string {
	if target.Scope == platform.ScopeGlobal {
		return filepath.Join(vscodeUserDir(target.Root), "mcp.json")
	}
	return target.Rel(".vscode", "mcp.json")
}

// vscodeUserDir returns the directory VS Code keeps per-user settings in on the
// current platform.
func vscodeUserDir(home string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Code", "User")
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "Code", "User")
		}
		return filepath.Join(home, "AppData", "Roaming", "Code", "User")
	default:
		if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
			return filepath.Join(configHome, "Code", "User")
		}
		return filepath.Join(home, ".config", "Code", "User")
	}
}

// serverEntry builds VS Code's entry. Its MCP framework requires the
// transport type on both forms, and silently ignores an entry without one.
func serverEntry(opts common.Options) map[string]any {
	if common.IsHTTP(opts) {
		return common.AddType(common.ManagedHTTP(opts, "url"), common.TransportHTTP)
	}
	return common.AddType(common.ManagedServer(opts), common.TransportStdio)
}

// upsertMCP merges the managed entry into the VS Code configuration. VS Code
// names its server table "servers" and long shipped "mcpServers", so an older
// file is migrated onto the current key as it is written.
func upsertMCP(path string, opts common.Options) (bool, string, error) {
	return common.UpsertMCPServerInKeyReplacingLegacy(
		path,
		settingsLabel,
		"servers",
		opts.ServerName,
		serverEntry(opts),
		nil,
		opts,
	)
}

func removeMCP(path string, opts common.Options) (bool, string, error) {
	return common.RemoveMCPServerInKey(path, settingsLabel, "servers", common.EntryName(opts), opts)
}
