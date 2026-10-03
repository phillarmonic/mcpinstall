package cursor

import (
	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

const settingsLabel = "Cursor MCP settings"

// mcpPath returns the Cursor file holding MCP servers for the target scope.
// Both scopes use the same relative location, so the target root alone decides
// whether the entry lands in the project or in the home directory.
func mcpPath(target platform.Target) string {
	return target.Rel(".cursor", "mcp.json")
}

func upsertMCP(path string, opts common.Options) (bool, string, error) {
	return common.UpsertMCPServer(path, settingsLabel, opts.ServerName, common.ManagedServer(opts), opts)
}

func removeMCP(path string, opts common.Options) (bool, string, error) {
	return common.RemoveMCPServer(path, settingsLabel, common.EntryName(opts), opts)
}
