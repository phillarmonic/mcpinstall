// Package copilot distributes the MCP server into the
// configuration shared by GitHub Copilot in VS Code.
package copilot

import (
	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

// Adapter implements platform.Adapter for Copilot in VS Code.
type Adapter struct{}

// Name is the catalog name used to select this adapter.
func (Adapter) Name() string { return "copilot" }

// Status reports that VS Code has a machine-readable MCP configuration. The
// server table is named "servers", unlike the "mcpServers" key the other hosts
// use.
func (Adapter) Status() platform.AdapterStatus {
	return platform.AdapterStatus{
		Supported: true,
		Capabilities: platform.Capabilities{
			MCP:        true,
			Format:     "json",
			ServersKey: "servers",
		},
	}
}

// Install merges the managed server entry into the VS Code configuration.
func (Adapter) Install(target platform.Target, opts common.Options) ([]string, []string, error) {
	path := mcpPath(target)
	changed, warning, err := upsertMCP(path, opts)
	if err != nil {
		return nil, nil, err
	}
	return common.Paths(path, changed), common.Warnings(warning), nil
}

// Remove deletes the managed server entry from the VS Code configuration.
func (Adapter) Remove(target platform.Target, opts common.Options) ([]string, []string, error) {
	path := mcpPath(target)
	changed, warning, err := removeMCP(path, opts)
	if err != nil {
		return nil, nil, err
	}
	return common.Paths(path, changed), common.Warnings(warning), nil
}
