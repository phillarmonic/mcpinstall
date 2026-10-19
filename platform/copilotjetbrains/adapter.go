// Package copilotjetbrains distributes the MCP server into the
// configuration GitHub Copilot for JetBrains reads. The plugin keeps its own
// user-level file, separate from the one Copilot in VS Code uses, so it needs
// an adapter of its own rather than sharing the copilot package.
package copilotjetbrains

import (
	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

// Adapter implements platform.Adapter for Copilot in JetBrains IDEs.
type Adapter struct{}

// Name is the catalog name used to select this adapter.
func (Adapter) Name() string { return "copilot-jetbrains" }

// SupportsScope reports that the plugin has only a user-level file. A
// project-scope run skips it instead of failing every other host.
func (Adapter) SupportsScope(scope platform.Scope) bool {
	return scope == platform.ScopeGlobal
}

// Status reports that the JetBrains plugin has a machine-readable MCP
// configuration. Like VS Code it names its server table "servers", unlike the
// "mcpServers" key the other hosts use.
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

// Install merges the managed server entry into the JetBrains configuration.
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

// Remove deletes the managed server entry from the JetBrains configuration.
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
