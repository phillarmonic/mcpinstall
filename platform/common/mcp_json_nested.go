package common

import (
	"bytes"
	"fmt"
	"os"
)

// This file extends the JSON merge rules to hosts that keep their server map
// below a nested key, such as OpenClaw's mcp.servers. The rules are the same
// as the flat writers: a malformed file is preserved with a warning, an
// existing entry is left alone unless forced, and emptying the map prunes the
// containers behind it.

// UpsertMCPServerAtPath merges serverName into the object at keys under the
// JSON file at path, creating missing containers. A container that exists but
// is not an object is preserved with a warning.
func UpsertMCPServerAtPath(path, label string, keys []string, serverName string, server map[string]any, opts Options) (bool, string, error) {
	cfg, malformed, err := readJSONObject(path)
	if err != nil {
		return false, "", err
	}
	if malformed {
		return false, fmt.Sprintf("preserved existing %s %s because the file is not valid JSON", label, path), nil
	}

	servers, warning := ensureObjectPath(cfg, keys)
	if warning != "" {
		return false, fmt.Sprintf("preserved existing %s %s because %s", label, path, warning), nil
	}
	if _, exists := servers[serverName]; exists && !opts.Force {
		return false, "", nil
	}
	servers[serverName] = server
	return WriteJSON(path, cfg, opts)
}

// RemoveMCPServerAtPath deletes serverName from the object at keys under the
// JSON file at path. Containers the removal empties are pruned, and the file is
// deleted once nothing remains.
func RemoveMCPServerAtPath(path, label string, keys []string, serverName string, opts Options) (bool, string, error) {
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

	servers, containers, found, warning := lookupObjectPath(cfg, keys)
	if warning != "" {
		return false, fmt.Sprintf("preserved existing %s %s during uninstall because %s", label, path, warning), nil
	}
	if !found {
		return false, "", nil
	}
	if _, exists := servers[serverName]; !exists {
		return false, "", nil
	}
	delete(servers, serverName)

	// Prune empty maps from the deepest key outward.
	for index := len(keys) - 1; index >= 0; index-- {
		if len(containers[index+1]) != 0 {
			break
		}
		delete(containers[index], keys[index])
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

// ensureObjectPath returns the object at keys, creating missing containers.
// A warning is returned when an existing value along the path is not an object.
func ensureObjectPath(root map[string]any, keys []string) (map[string]any, string) {
	current := root
	for _, key := range keys {
		value, exists := current[key]
		if !exists {
			next := map[string]any{}
			current[key] = next
			current = next
			continue
		}
		next, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Sprintf("%s is not a JSON object", key)
		}
		current = next
	}
	return current, ""
}

// lookupObjectPath returns the object at keys plus the chain of containers
// leading to it, so a removal can prune what it empties.
func lookupObjectPath(root map[string]any, keys []string) (map[string]any, []map[string]any, bool, string) {
	containers := []map[string]any{root}
	current := root
	for _, key := range keys {
		value, exists := current[key]
		if !exists {
			return nil, containers, false, ""
		}
		next, ok := value.(map[string]any)
		if !ok {
			return nil, containers, false, fmt.Sprintf("%s is not a JSON object", key)
		}
		current = next
		containers = append(containers, current)
	}
	return current, containers, true, ""
}
