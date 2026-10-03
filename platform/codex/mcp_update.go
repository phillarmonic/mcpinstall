package codex

import (
	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

const settingsLabel = "Codex MCP settings"

// mcpPath returns the Codex configuration holding MCP servers for the target
// scope. Both scopes use the same relative location, so the target root alone
// decides whether the entry lands in the project or in the home directory.
func mcpPath(target platform.Target) string {
	return target.Rel(".codex", "config.toml")
}

// managedMCPTemplate builds the Codex server table. Codex approves MCP tool
// calls interactively by default, which would prompt on every MCP server
// call, so the managed entry opts into auto-approval.
func managedMCPTemplate(opts common.Options) map[string]any {
	server := common.ManagedServer(opts)
	server["default_tools_approval_mode"] = "approve"
	return server
}

func upsertMCP(path string, opts common.Options) (bool, string, error) {
	return common.UpsertTOMLMCPServer(path, settingsLabel, "mcp_servers", opts.ServerName, managedMCPTemplate(opts), opts)
}

func removeMCP(path string, opts common.Options) (bool, string, error) {
	return common.RemoveTOMLMCPServer(path, settingsLabel, "mcp_servers", common.EntryName(opts), opts)
}
