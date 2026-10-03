package common

import (
	"bytes"
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// This is the TOML counterpart of mcp_json.go, for hosts such as Codex that
// configure MCP servers in TOML. The merge rules are deliberately identical:
// malformed files are preserved with a warning, an entry that already exists is
// left alone, and emptying the table cleans up the containers behind it.

// UpsertTOMLMCPServer merges serverName into the serversKey table of the TOML
// file at path, creating the file when it does not exist.
func UpsertTOMLMCPServer(path, label, serversKey, serverName string, server map[string]any, opts Options) (bool, string, error) {
	cfg, malformed, err := readTOMLObject(path)
	if err != nil {
		return false, "", err
	}
	if malformed {
		return false, fmt.Sprintf("preserved existing %s %s because the file is not valid TOML", label, path), nil
	}

	serversValue, ok := cfg[serversKey]
	if !ok {
		cfg[serversKey] = map[string]any{serverName: server}
		return WriteTOML(path, cfg, opts)
	}

	serversMap, ok := toStringMap(serversValue)
	if !ok {
		return false, fmt.Sprintf("preserved existing %s %s because %s is not a TOML table", label, path, serversKey), nil
	}

	if _, exists := serversMap[serverName]; exists && !opts.Force {
		return false, "", nil
	}
	serversMap[serverName] = server
	return WriteTOML(path, cfg, opts)
}

// RemoveTOMLMCPServer deletes serverName from the serversKey table of the TOML
// file at path, dropping emptied containers and the file itself as it goes.
func RemoveTOMLMCPServer(path, label, serversKey, serverName string, opts Options) (bool, string, error) {
	// #nosec G304 -- path is a host configuration location derived from the
	// resolved scope root; reading it is the purpose of this package.
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, "", nil
		}
		return false, "", err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return false, "", nil
	}

	cfg, malformed := parseTOMLObject(data)
	if malformed {
		return false, fmt.Sprintf("preserved existing %s %s during uninstall because the file is not valid TOML", label, path), nil
	}

	serversMap, ok := toStringMap(cfg[serversKey])
	if !ok {
		return false, "", nil
	}
	if _, exists := serversMap[serverName]; !exists {
		return false, "", nil
	}
	delete(serversMap, serverName)

	if len(serversMap) == 0 {
		delete(cfg, serversKey)
	}
	if len(cfg) == 0 {
		if opts.DryRun {
			return true, "", nil
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return false, "", err
		}
		return true, "", nil
	}
	return WriteTOML(path, cfg, opts)
}

// readTOMLObject loads path as a TOML table. The bool reports whether the file
// held content that could not be parsed, which callers degrade to a warning so
// a malformed config is never clobbered.
func readTOMLObject(path string) (map[string]any, bool, error) {
	// #nosec G304 -- see RemoveTOMLMCPServer.
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, false, nil
		}
		return nil, false, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return map[string]any{}, false, nil
	}
	cfg, malformed := parseTOMLObject(data)
	return cfg, malformed, nil
}

// parseTOMLObject decodes a TOML table, reporting whether the input was
// malformed instead of failing.
func parseTOMLObject(data []byte) (map[string]any, bool) {
	cfg := map[string]any{}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, true
	}
	return cfg, false
}

// marshalTOML renders a TOML document for the shared write path.
func marshalTOML(value any) ([]byte, error) {
	return toml.Marshal(value)
}

// toStringMap normalises the map types the TOML decoder produces so callers can
// mutate a nested table in place.
func toStringMap(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	case map[string]string:
		converted := make(map[string]any, len(typed))
		for key, item := range typed {
			converted[key] = item
		}
		return converted, true
	default:
		return nil, false
	}
}
