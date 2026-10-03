// Package opencode distributes the MCP server into OpenCode's
// configuration.
package opencode

import (
	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

// Adapter implements platform.Adapter for OpenCode.
type Adapter struct{}

// Name is the catalog name used to select this adapter.
func (Adapter) Name() string { return "opencode" }

// Status reports that OpenCode has a machine-readable MCP configuration.
func (Adapter) Status() platform.AdapterStatus {
	return platform.AdapterStatus{
		Supported: true,
		Capabilities: platform.Capabilities{
			MCP:        true,
			Format:     "json",
			ServersKey: serversKey,
		},
	}
}

// Install merges the managed server entry into the OpenCode configuration.
func (Adapter) Install(target platform.Target, opts common.Options) ([]string, []string, error) {
	path, err := mcpPath(target)
	if err != nil {
		return nil, nil, err
	}
	changed, warning, err := upsertMCP(path, opts)
	if err != nil {
		return nil, nil, err
	}
	return common.Paths(path, changed), common.Warnings(warning), nil
}

// Remove deletes the managed server entry from the OpenCode configuration.
func (Adapter) Remove(target platform.Target, opts common.Options) ([]string, []string, error) {
	path, err := mcpPath(target)
	if err != nil {
		return nil, nil, err
	}
	changed, warning, err := removeMCP(path, opts)
	if err != nil {
		return nil, nil, err
	}
	return common.Paths(path, changed), common.Warnings(warning), nil
}
