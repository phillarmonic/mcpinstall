package claude

import (
	"os"
	"path/filepath"

	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

const settingsLabel = "Claude MCP settings"

// mcpPath returns the Claude Code file holding MCP servers for the target
// scope. Project scope uses the dedicated .mcp.json; user scope keeps MCP
// servers alongside the rest of Claude Code's user state in ~/.claude.json.
func mcpPath(target platform.Target) string {
	if target.Scope == platform.ScopeGlobal {
		// CLAUDE_CONFIG_DIR relocates Claude Code's whole config home,
		// including the .claude.json that holds MCP servers.
		if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
			return filepath.Join(dir, ".claude.json")
		}
		return target.Rel(".claude.json")
	}
	return target.Rel(".mcp.json")
}

// serverEntry builds Claude Code's entry. A stdio server keeps the plain
// command/args shape Claude Code has always accepted; a remote one names its
// transport explicitly, which is what its HTTP reader requires.
func serverEntry(opts common.Options) map[string]any {
	if common.IsHTTP(opts) {
		return common.AddType(common.ManagedHTTP(opts, "url"), common.TransportHTTP)
	}
	return common.ManagedServer(opts)
}

func upsertMCP(path string, opts common.Options) (bool, string, error) {
	return common.UpsertMCPServer(path, settingsLabel, opts.ServerName, serverEntry(opts), opts)
}

func removeMCP(path string, opts common.Options) (bool, string, error) {
	return common.RemoveMCPServer(path, settingsLabel, common.EntryName(opts), opts)
}
