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

// serverEntry builds Cursor's entry: a plain stdio command, or the url and
// headers shape its remote MCP reader takes.
func serverEntry(opts common.Options) map[string]any {
	if common.IsHTTP(opts) {
		return common.ManagedHTTP(opts, "url")
	}
	return common.ManagedServer(opts)
}

func upsertMCP(path string, opts common.Options) (bool, string, error) {
	return common.UpsertMCPServer(path, settingsLabel, opts.ServerName, serverEntry(opts), opts)
}

func removeMCP(path string, opts common.Options) (bool, string, error) {
	return common.RemoveMCPServer(path, settingsLabel, common.EntryName(opts), opts)
}
