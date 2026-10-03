package claude

import (
	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

const settingsLabel = "Claude MCP settings"

// mcpPath returns the Claude Code file holding MCP servers for the target
// scope. Project scope uses the dedicated .mcp.json; user scope keeps MCP
// servers alongside the rest of Claude Code's user state in ~/.claude.json.
func mcpPath(target platform.Target) string {
	if target.Scope == platform.ScopeGlobal {
		return target.Rel(".claude.json")
	}
	return target.Rel(".mcp.json")
}

func upsertMCP(path string, opts common.Options) (bool, string, error) {
	return common.UpsertMCPServer(path, settingsLabel, opts.ServerName, common.ManagedServer(opts), opts)
}

func removeMCP(path string, opts common.Options) (bool, string, error) {
	return common.RemoveMCPServer(path, settingsLabel, common.EntryName(opts), opts)
}
