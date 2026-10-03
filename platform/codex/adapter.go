// Package codex distributes the MCP server into Codex's
// configuration, which is TOML rather than JSON.
package codex

import (
	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

// Adapter implements platform.Adapter for Codex.
type Adapter struct{}

// Name is the catalog name used to select this adapter.
func (Adapter) Name() string { return "codex" }

// Status reports that Codex has a machine-readable MCP configuration written in
// TOML under an "mcp_servers" table.
func (Adapter) Status() platform.AdapterStatus {
	return platform.AdapterStatus{
		Supported: true,
		Capabilities: platform.Capabilities{
			MCP:        true,
			Format:     "toml",
			ServersKey: "mcp_servers",
		},
	}
}

// Install merges the managed server table into the Codex configuration.
func (Adapter) Install(target platform.Target, opts common.Options) ([]string, []string, error) {
	path := mcpPath(target)
	changed, warning, err := upsertMCP(path, opts)
	if err != nil {
		return nil, nil, err
	}
	return common.Paths(path, changed), common.Warnings(warning), nil
}

// Remove deletes the managed server table from the Codex configuration.
func (Adapter) Remove(target platform.Target, opts common.Options) ([]string, []string, error) {
	path := mcpPath(target)
	changed, warning, err := removeMCP(path, opts)
	if err != nil {
		return nil, nil, err
	}
	return common.Paths(path, changed), common.Warnings(warning), nil
}
