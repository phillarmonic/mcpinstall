// Package claude distributes the MCP server into Claude Code's
// configuration.
package claude

import (
	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

// Adapter implements platform.Adapter for Claude Code.
type Adapter struct{}

// Name is the catalog name used to select this adapter.
func (Adapter) Name() string { return "claude" }

// Status reports that Claude Code has a machine-readable MCP configuration.
func (Adapter) Status() platform.AdapterStatus {
	return platform.AdapterStatus{
		Supported: true,
		Capabilities: platform.Capabilities{
			MCP:        true,
			Format:     "json",
			ServersKey: "mcpServers",
		},
	}
}

// Install merges the managed server entry into the Claude Code configuration.
func (Adapter) Install(target platform.Target, opts common.Options) ([]string, []string, error) {
	path := mcpPath(target)
	changed, warning, err := upsertMCP(path, opts)
	if err != nil {
		return nil, nil, err
	}
	return common.Paths(path, changed), common.Warnings(warning), nil
}

// Remove deletes the managed server entry from the Claude Code configuration.
func (Adapter) Remove(target platform.Target, opts common.Options) ([]string, []string, error) {
	path := mcpPath(target)
	changed, warning, err := removeMCP(path, opts)
	if err != nil {
		return nil, nil, err
	}
	return common.Paths(path, changed), common.Warnings(warning), nil
}
