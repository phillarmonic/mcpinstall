// Package platform defines the host abstraction used to distribute MCP
// configuration. An Adapter knows where one host keeps its MCP servers and how
// to merge the MCP server into that file without disturbing the rest of
// the user's settings.
//
// The shape mirrors the adapter/catalog layout used by comb-os: one small
// package per host, a shared writer for the merge itself, and a catalog that
// resolves user-supplied host names onto adapters.
package platform

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/phillarmonic/mcpinstall/platform/common"
)

// Scope selects which configuration file a host adapter reads and writes.
type Scope string

const (
	// ScopeProject targets configuration that lives inside the current project.
	ScopeProject Scope = "project"
	// ScopeGlobal targets configuration that applies to every project.
	ScopeGlobal Scope = "global"
)

// ParseScope maps a user-supplied scope name onto a Scope. An empty value
// selects ScopeProject, which is the least surprising default because it only
// touches the current repository.
func ParseScope(name string) (Scope, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", string(ScopeProject):
		return ScopeProject, nil
	case string(ScopeGlobal):
		return ScopeGlobal, nil
	default:
		return "", fmt.Errorf("unsupported scope %q (supported: %s, %s)", name, ScopeProject, ScopeGlobal)
	}
}

// Target is the base location an adapter resolves its configuration against.
type Target struct {
	Scope Scope
	// Root is the project directory for ScopeProject and the home directory
	// for ScopeGlobal.
	Root string
}

// Rel joins path elements onto the target root.
func (t Target) Rel(elem ...string) string {
	return filepath.Join(append([]string{t.Root}, elem...)...)
}

// Capabilities describes what a host adapter can configure.
//
// Field order is load-bearing for the garbage collector's pointer scan, so
// keep the strings ahead of the boolean rather than grouping for readability.
type Capabilities struct {
	// Format is the configuration syntax: "json" or "toml".
	Format string `json:"format"`
	// ServersKey is the configuration key holding the server table.
	ServersKey string `json:"servers_key"`
	// MCP reports whether the host has a machine-readable MCP configuration.
	MCP bool `json:"mcp"`
}

// AdapterStatus reports whether a host is supported and what it can do.
type AdapterStatus struct {
	Capabilities Capabilities `json:"capabilities"`
	Supported    bool         `json:"supported"`
}

// ScopeSupport is implemented by an adapter that cannot configure every
// scope. install.Run consults it before invoking the adapter, so a global-only
// host selected through "all" is skipped with a warning instead of failing the
// run.
type ScopeSupport interface {
	SupportsScope(scope Scope) bool
}

// Adapter distributes the MCP server entry into one host's
// configuration file.
type Adapter interface {
	// Name is the catalog name used to select this adapter.
	Name() string
	// Status reports support and capabilities for this host.
	Status() AdapterStatus
	// Install merges the managed server entry into the host configuration and
	// returns the paths it created or changed. Under a dry run it returns the
	// paths it would change and leaves the filesystem untouched.
	Install(target Target, opts common.Options) (created []string, warnings []string, err error)
	// Remove deletes the managed server entry from the host configuration and
	// returns the paths it removed or changed.
	Remove(target Target, opts common.Options) (removed []string, warnings []string, err error)
}
