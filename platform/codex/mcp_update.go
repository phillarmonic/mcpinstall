package codex

import (
	"os"
	"path/filepath"

	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

const settingsLabel = "Codex MCP settings"

// mcpPath returns the Codex configuration holding MCP servers for the target
// scope. Both scopes use the same relative location, so the target root alone
// decides whether the entry lands in the project or in the home directory.
func mcpPath(target platform.Target) string {
	// CODEX_HOME relocates Codex's whole config home, including the
	// config.toml that holds MCP servers.
	if dir := os.Getenv("CODEX_HOME"); dir != "" {
		return filepath.Join(dir, "config.toml")
	}
	return target.Rel(".codex", "config.toml")
}

// managedMCPTemplate builds the Codex server table. Codex approves MCP tool
// calls interactively by default, which would prompt on every MCP server
// call, so the managed entry opts into auto-approval.
//
// Codex draws a hard line between transports: an HTTP server takes `url` plus
// `http_headers` (or `bearer_token_env_var`), never a literal `bearer_token`.
func managedMCPTemplate(opts common.Options) map[string]any {
	var server map[string]any
	if common.IsHTTP(opts) {
		server = map[string]any{"url": opts.URL}
		if opts.TokenEnvVar != "" {
			server["bearer_token_env_var"] = opts.TokenEnvVar
		} else if headers := common.HeadersMap(opts); headers != nil {
			server["http_headers"] = headers
		}
	} else {
		server = common.ManagedServer(opts)
	}
	server["default_tools_approval_mode"] = "approve"
	return server
}

func upsertMCP(path string, opts common.Options) (bool, string, error) {
	return common.UpsertTOMLMCPServer(path, settingsLabel, "mcp_servers", opts.ServerName, managedMCPTemplate(opts), opts)
}

func removeMCP(path string, opts common.Options) (bool, string, error) {
	return common.RemoveTOMLMCPServer(path, settingsLabel, "mcp_servers", common.EntryName(opts), opts)
}
