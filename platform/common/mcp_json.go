package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
)

// This file mirrors the merge semantics comb-os uses for JSON host configs.
// The rules exist so an install never damages a file the user also maintains:
//
//   - A file that is not valid JSON is preserved untouched and reported as a
//     warning rather than overwritten.
//   - An entry that is already present is left alone unless it matches a known
//     legacy template (an upgrade) or the caller asked to force.
//   - Removing the managed entry cleans up the containers it emptied, and drops
//     the file entirely once nothing managed remains.

// legacyKeyMCP is the server table every host used before some of them renamed
// it. Only a host that really did rename its key may adopt it.
const legacyKeyMCP = "mcpServers"

// UpsertMCPServer merges serverName into the "mcpServers" object of the JSON
// file at path, creating the file when it does not exist.
func UpsertMCPServer(path, label, serverName string, server map[string]any, opts Options) (bool, string, error) {
	return UpsertMCPServerInKey(path, label, legacyKeyMCP, serverName, server, opts)
}

// UpsertMCPServerInKey merges serverName into the serversKey object of the JSON
// file at path. Hosts that do not use "mcpServers" pass their own key.
func UpsertMCPServerInKey(path, label, serversKey, serverName string, server map[string]any, opts Options) (bool, string, error) {
	return UpsertMCPServerInKeyReplacingLegacy(path, label, serversKey, serverName, server, nil, opts)
}

// UpsertMCPServerInKeyReplacingLegacy merges serverName into the serversKey
// object of the JSON file at path, migrating a legacy "mcpServers" object onto
// serversKey when one is found.
//
// legacyServer lets a host upgrade an entry this tool wrote in an earlier
// version: when the existing entry is byte-for-byte the legacy template it is
// replaced with the current one. A user-customised entry never matches and is
// therefore preserved.
func UpsertMCPServerInKeyReplacingLegacy(path, label, serversKey, serverName string, server, legacyServer map[string]any, opts Options) (bool, string, error) {
	return upsertInKey(path, label, serversKey, legacyKeyMCP, serverName, server, legacyServer, opts)
}

// UpsertMCPServerInKeyOnly merges serverName into the serversKey object of the
// JSON file at path without adopting a legacy key.
//
// A host whose configuration has only ever used serversKey passes this: an
// unrelated "mcpServers" object is left exactly as the user wrote it, rather
// than being renamed onto a key that would suddenly make those servers live.
func UpsertMCPServerInKeyOnly(path, label, serversKey, serverName string, server map[string]any, opts Options) (bool, string, error) {
	return upsertInKey(path, label, serversKey, "", serverName, server, nil, opts)
}

// upsertInKey merges serverName into the serversKey object of the JSON file at
// path. legacyKey names a second key to adopt when serversKey is absent, so a
// host that really did rename its table can be migrated; an empty legacyKey
// leaves every other key in the file alone.
func upsertInKey(path, label, serversKey, legacyKey, serverName string, server, legacyServer map[string]any, opts Options) (bool, string, error) {
	cfg, malformed, err := readJSONObject(path)
	if err != nil {
		return false, "", err
	}
	if malformed {
		return false, fmt.Sprintf("preserved existing %s %s because the file is not valid JSON", label, path), nil
	}

	migrated := false
	serversValue, ok := cfg[serversKey]
	if !ok && legacyKey != "" && legacyKey != serversKey {
		if legacy, hasLegacy := cfg[legacyKey]; hasLegacy {
			cfg[serversKey] = legacy
			delete(cfg, legacyKey)
			serversValue = legacy
			ok = true
			migrated = true
		}
	}
	if !ok {
		cfg[serversKey] = map[string]any{serverName: server}
		return WriteJSON(path, cfg, opts)
	}

	serversMap, ok := serversValue.(map[string]any)
	if !ok {
		return false, fmt.Sprintf("preserved existing %s %s because %s is not a JSON object", label, path, serversKey), nil
	}

	if existing, exists := serversMap[serverName]; exists {
		switch {
		case legacyServer != nil && reflect.DeepEqual(existing, legacyServer):
			serversMap[serverName] = server
		case opts.Force:
			serversMap[serverName] = server
		case migrated:
			// Persist the key migration even though the entry itself is current.
		default:
			return false, "", nil
		}
		return WriteJSON(path, cfg, opts)
	}

	serversMap[serverName] = server
	return WriteJSON(path, cfg, opts)
}

// RemoveMCPServer deletes serverName from the "mcpServers" object of the JSON
// file at path.
func RemoveMCPServer(path, label, serverName string, opts Options) (bool, string, error) {
	return RemoveMCPServerInKey(path, label, legacyKeyMCP, serverName, opts)
}

// RemoveMCPServerInKey deletes serverName from the serversKey object of the
// JSON file at path, falling back to "mcpServers" when serversKey is absent.
// Containers that the removal empties are dropped, and the file itself is
// deleted once the configuration has nothing left in it.
func RemoveMCPServerInKey(path, label, serversKey, serverName string, opts Options) (bool, string, error) {
	return removeFromKey(path, label, serversKey, legacyKeyMCP, serverName, opts)
}

// RemoveMCPServerInKeyOnly deletes serverName from the serversKey object of the
// JSON file at path without falling back to a legacy key, so a host whose
// configuration has only ever used serversKey never has an unrelated object
// touched. It follows the same container-cleanup rules as RemoveMCPServerInKey.
func RemoveMCPServerInKeyOnly(path, label, serversKey, serverName string, opts Options) (bool, string, error) {
	return removeFromKey(path, label, serversKey, "", serverName, opts)
}

// removeFromKey deletes serverName from the serversKey object of the JSON file
// at path, falling back to legacyKey when serversKey is absent so a host that
// renamed its table is cleaned up in the place it was written. An empty
// legacyKey leaves every other key in the file alone.
func removeFromKey(path, label, serversKey, legacyKey, serverName string, opts Options) (bool, string, error) {
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

	cfg, malformed := parseJSONObject(data)
	if malformed {
		return false, fmt.Sprintf("preserved existing %s %s during uninstall because the file is not valid JSON", label, path), nil
	}

	serversMap, ok := cfg[serversKey].(map[string]any)
	activeKey := serversKey
	if !ok && legacyKey != "" && legacyKey != serversKey {
		serversMap, ok = cfg[legacyKey].(map[string]any)
		activeKey = legacyKey
	}
	if !ok {
		return false, "", nil
	}
	if _, exists := serversMap[serverName]; !exists {
		return false, "", nil
	}
	delete(serversMap, serverName)

	if len(serversMap) == 0 {
		delete(cfg, activeKey)
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
	return WriteJSON(path, cfg, opts)
}

// readJSONObject loads path as a JSON object. The bool reports whether the file
// held content that could not be parsed, which callers degrade to a warning so
// a malformed config is never clobbered.
func readJSONObject(path string) (map[string]any, bool, error) {
	// #nosec G304 -- see RemoveMCPServerInKey.
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
	cfg, malformed := parseJSONObject(data)
	return cfg, malformed, nil
}

// parseJSONObject decodes a JSON object, reporting whether the input was
// malformed instead of failing. JSONC input — the shape several hosts ship —
// is normalised first, so comments and trailing commas are not mistaken for a
// malformed configuration.
func parseJSONObject(data []byte) (map[string]any, bool) {
	cfg := map[string]any{}
	if err := json.Unmarshal(normalizeJSONC(data), &cfg); err != nil {
		return nil, true
	}
	return cfg, false
}
