// Package hosts holds the adapters for the agent hosts beyond the original six.
// The host knowledge is ported from ai-memory's install-mcp client matrix: each
// descriptor names one host's configuration path, its server table, and the
// entry shape that host documents.
//
// A descriptor is data rather than a hand-written adapter because most hosts
// differ only in those three facts. The custom cases (Muse's dual key,
// OpenCode's nested V2 map, swival's project-root probe) are still expressed
// here, but they share the same merge machinery as everything else.
package hosts

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

// Kind selects the writer a descriptor uses.
type Kind int

const (
	// JSONTop writes a server map directly under Key, e.g. mcpServers.
	JSONTop Kind = iota
	// JSONNested writes a server map below a dotted Key, e.g. mcp.servers.
	JSONNested
	// TOMLTop writes a server table under Key, e.g. mcp_servers.
	TOMLTop
	// Muse writes to whichever of mcpServers/mcp_servers the file already uses.
	Muse
)

// Descriptor is one host adapter expressed as data.
type Descriptor struct {
	// Host is the catalog name.
	Host string
	// Label names the file in warnings.
	Label string
	// Capabilities describes the host's configuration format and server key.
	Capabilities platform.Capabilities
	// Scopes lists the scopes the host supports. Empty means both.
	Scopes []platform.Scope
	// Path resolves the configuration file for the target scope.
	Path func(platform.Target) (string, error)
	// Kind selects the writer.
	Kind Kind
	// Key is the server map's key, dotted for JSONNested.
	Key string
	// LegacyKey migrates an older key onto Key when present.
	LegacyKey string
	// Validate rejects an identity the host cannot load. Nil accepts every
	// identity the shared validator already accepted.
	Validate func(common.Options) error
	// Entry builds the host-specific server entry.
	Entry func(common.Options) (map[string]any, error)
	// Unsupported, when set, explains why the host has no writer. It is
	// reported by an explicit selection and skipped by "all".
	Unsupported string
}

// Name is the catalog name.
func (d Descriptor) Name() string { return d.Host }

// Status reports support and capabilities.
func (d Descriptor) Status() platform.AdapterStatus {
	return platform.AdapterStatus{
		Supported:    d.Unsupported == "",
		Capabilities: d.Capabilities,
	}
}

// SupportsScope reports whether the host can write this scope.
func (d Descriptor) SupportsScope(scope platform.Scope) bool {
	// An unsupported host still answers true so an explicit selection reaches
	// Install and gets its guidance instead of a misleading scope warning.
	if d.Unsupported != "" {
		return true
	}
	if len(d.Scopes) == 0 {
		return true
	}
	for _, supported := range d.Scopes {
		if supported == scope {
			return true
		}
	}
	return false
}

// Install merges the managed entry into the host configuration.
func (d Descriptor) Install(target platform.Target, opts common.Options) ([]string, []string, error) {
	if d.Unsupported != "" {
		return nil, nil, errors.New(d.Unsupported)
	}
	path, err := d.Path(target)
	if err != nil {
		return nil, nil, err
	}
	if d.Validate != nil {
		if err := d.Validate(opts); err != nil {
			return nil, nil, err
		}
	}
	entry, err := d.Entry(opts)
	if err != nil {
		// A host that cannot express this transport is skipped with a
		// warning, so selecting "all" does not fail on a single such host.
		if errors.Is(err, errZCodeNeedsHTTP) {
			return nil, []string{err.Error()}, nil
		}
		return nil, nil, err
	}

	var (
		changed bool
		warning string
	)
	switch d.Kind {
	case JSONTop:
		if d.LegacyKey != "" {
			changed, warning, err = common.UpsertMCPServerInKeyReplacingLegacy(
				path, d.Label, d.Key, opts.ServerName, entry, nil, opts,
			)
		} else {
			changed, warning, err = common.UpsertMCPServerInKeyOnly(
				path, d.Label, d.Key, opts.ServerName, entry, opts,
			)
		}
	case JSONNested:
		changed, warning, err = common.UpsertMCPServerAtPath(
			path, d.Label, strings.Split(d.Key, "."), opts.ServerName, entry, opts,
		)
	case TOMLTop:
		changed, warning, err = common.UpsertTOMLMCPServer(
			path, d.Label, d.Key, opts.ServerName, entry, opts,
		)
	case Muse:
		changed, warning, err = upsertMuse(path, d.Label, opts, entry)
	default:
		err = fmt.Errorf("%s: unknown writer kind", d.Host)
	}
	if err != nil {
		return nil, nil, err
	}
	return common.Paths(path, changed), common.Warnings(warning), nil
}

// Remove deletes the managed entry from the host configuration.
func (d Descriptor) Remove(target platform.Target, opts common.Options) ([]string, []string, error) {
	if d.Unsupported != "" {
		return nil, nil, errors.New(d.Unsupported)
	}
	path, err := d.Path(target)
	if err != nil {
		return nil, nil, err
	}
	name := common.EntryName(opts)

	var (
		changed bool
		warning string
	)
	switch d.Kind {
	case JSONTop:
		if d.LegacyKey != "" {
			changed, warning, err = common.RemoveMCPServerInKey(path, d.Label, d.Key, name, opts)
		} else {
			changed, warning, err = common.RemoveMCPServerInKeyOnly(path, d.Label, d.Key, name, opts)
		}
	case JSONNested:
		changed, warning, err = common.RemoveMCPServerAtPath(
			path, d.Label, strings.Split(d.Key, "."), name, opts,
		)
	case TOMLTop:
		changed, warning, err = common.RemoveTOMLMCPServer(path, d.Label, d.Key, name, opts)
	case Muse:
		changed, warning, err = removeMuse(path, d.Label, name, opts)
	default:
		err = fmt.Errorf("%s: unknown writer kind", d.Host)
	}
	if err != nil {
		return nil, nil, err
	}
	return common.Paths(path, changed), common.Warnings(warning), nil
}

// All returns every host adapter keyed by catalog name.
func All() map[string]platform.Adapter {
	adapters := make(map[string]platform.Adapter, len(descriptors))
	for _, descriptor := range descriptors {
		adapters[descriptor.Host] = descriptor
	}
	return adapters
}

// upsertMuse writes Muse's entry to the server map the file already uses. Muse
// drops its whole MCP block when both mcpServers and mcp_servers are present,
// so exactly one key is ever written. It also requires schema_version, and
// fails every command at startup without it.
func upsertMuse(path, label string, opts common.Options, entry map[string]any) (bool, string, error) {
	root, malformed, err := common.ReadJSONObject(path)
	if err != nil {
		return false, "", err
	}
	if malformed {
		return false, fmt.Sprintf("preserved existing %s %s because the file is not valid JSON", label, path), nil
	}

	key, err := museKey(root)
	if err != nil {
		return false, "", err
	}
	entry = museEntryForKey(entry, key)

	servers, ok := root[key].(map[string]any)
	if !ok && root[key] != nil {
		return false, fmt.Sprintf("preserved existing %s %s because %s is not a JSON object", label, path, key), nil
	}
	if servers == nil {
		servers = map[string]any{}
		root[key] = servers
	}

	if _, exists := servers[opts.ServerName]; exists && !opts.Force {
		if _, hasSchema := root["schema_version"]; hasSchema {
			return false, "", nil
		}
		root["schema_version"] = 1
		return common.WriteJSON(path, root, opts)
	}
	servers[opts.ServerName] = entry
	if _, hasSchema := root["schema_version"]; !hasSchema {
		root["schema_version"] = 1
	}
	return common.WriteJSON(path, root, opts)
}

// removeMuse deletes the entry and prunes the map it emptied. A file this
// installer created holds only schema_version afterwards, so that file is
// removed too rather than leaving a stub that Muse would try to load.
func removeMuse(path, label, name string, opts common.Options) (bool, string, error) {
	root, malformed, err := common.ReadJSONObject(path)
	if err != nil {
		return false, "", err
	}
	if malformed {
		return false, fmt.Sprintf("preserved existing %s %s during uninstall because the file is not valid JSON", label, path), nil
	}
	key, err := museKey(root)
	if err != nil {
		return false, "", err
	}
	servers, ok := root[key].(map[string]any)
	if !ok {
		return false, "", nil
	}
	if _, exists := servers[name]; !exists {
		return false, "", nil
	}
	delete(servers, name)
	if len(servers) == 0 {
		delete(root, key)
	}
	if len(root) == 1 {
		if _, onlySchema := root["schema_version"]; onlySchema {
			if opts.DryRun {
				return true, "", nil
			}
			if err := removeFile(path); err != nil {
				return false, "", err
			}
			return true, "", nil
		}
	}
	return common.WriteJSON(path, root, opts)
}

// removeFile deletes path, treating an already-absent file as success.
func removeFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// museKey picks the one server map Muse will read.
func museKey(root map[string]any) (string, error) {
	_, camel := root["mcpServers"]
	_, snake := root["mcp_servers"]
	if camel && snake {
		return "", errors.New("this Muse settings file has both mcpServers and mcp_servers, and Muse ignores every MCP server when both are present; move the servers into one map and re-run")
	}
	if snake {
		return "mcp_servers", nil
	}
	return "mcpServers", nil
}

// museEntryForKey renames the transport discriminator for the legacy
// snake_case map.
func museEntryForKey(entry map[string]any, key string) map[string]any {
	if key != "mcp_servers" {
		return entry
	}
	if kind, ok := entry["type"]; ok {
		delete(entry, "type")
		entry["transport"] = kind
	}
	return entry
}
