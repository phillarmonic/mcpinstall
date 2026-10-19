// Package common holds the shared config-writing machinery every host adapter
// builds on. Nothing here knows about a specific host, and nothing here imports
// the platform package, so adapters can depend on it freely.
package common

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
)

// Options carries the server identity and the decisions an install or
// uninstall run makes, threaded through to every adapter and config writer.
//
// Fields are ordered strings and slices before the booleans to keep the
// garbage collector's pointer scan tight.
type Options struct {
	// ServerName is the MCP server key written into every host configuration.
	ServerName string
	// Command is the executable hosts are told to launch for a stdio entry.
	Command string
	// Args are fixed arguments stored alongside the command.
	Args []string
	// Transport selects the entry shape: "stdio" or "http". Empty infers
	// "http" when URL is set, otherwise "stdio".
	Transport string
	// URL is the remote MCP endpoint used by HTTP entries.
	URL string
	// TokenEnvVar names an environment variable holding the bearer token for
	// hosts that support env-var indirection instead of a literal header.
	TokenEnvVar string
	// Headers are sent with every HTTP request. A host that cannot express a
	// literal header may use TokenEnvVar instead.
	Headers map[string]string
	// EntryName is the server key an uninstall deletes. Empty deletes
	// ServerName. Install ignores it.
	EntryName string
	// ToolMode selects how a host that groups MCP tools presents this server:
	// "code-mode" keeps the host grouping, "native" exposes each tool on the
	// provider's native tool list. Empty uses the host default.
	ToolMode string
	// Force overwrites a managed entry that is already present. Without it an
	// existing entry is left exactly as the user left it.
	Force bool
	// DryRun computes pending changes without touching the filesystem.
	DryRun bool
}

// WriteJSON marshals value as indented JSON with a trailing newline and writes
// it to path. It reports whether the file changed; an identical file is left
// untouched so repeated installs are quiet. Parent directories are created as
// needed. Under a dry run nothing is written but the change is still reported.
func WriteJSON(path string, value any, opts Options) (bool, string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return false, "", err
	}
	data = append(data, '\n')
	return writeFile(path, data, opts)
}

// WriteTOML marshals value as TOML and writes it to path, following the same
// change-reporting and dry-run rules as WriteJSON.
func WriteTOML(path string, value any, opts Options) (bool, string, error) {
	data, err := marshalTOML(value)
	if err != nil {
		return false, "", err
	}
	return writeFile(path, data, opts)
}

// writeFile writes data to path unless the file already holds exactly that
// content.
//
// The write goes through a temporary file in the same directory and a rename,
// so a host configuration that also holds unrelated state (Claude Code keeps a
// large shared file under the home directory, for instance) can never be left
// half-written if the process dies mid-write.
func writeFile(path string, data []byte, opts Options) (bool, string, error) {
	// #nosec G304 -- path is a host configuration location derived from the
	// resolved scope root; reading it is the purpose of this package.
	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, data) {
		return false, "", nil
	} else if err != nil && !os.IsNotExist(err) {
		return false, "", err
	}

	if opts.DryRun {
		return true, "", nil
	}
	dir := filepath.Dir(path)
	if mkErr := os.MkdirAll(dir, 0o750); mkErr != nil {
		return false, "", mkErr
	}

	tmp, err := os.CreateTemp(dir, ".aekf-*.tmp")
	if err != nil {
		return false, "", err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // no-op once the rename has landed

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return false, "", err
	}
	if err := tmp.Close(); err != nil {
		return false, "", err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return false, "", err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return false, "", err
	}
	return true, "", nil
}

// WriteText writes arbitrary text through the same atomic, idempotent path as
// WriteJSON and WriteTOML. It is what the skill writer uses for a managed file
// whose format belongs to another program.
func WriteText(path string, data []byte, opts Options) (bool, string, error) {
	return writeFile(path, data, opts)
}

// RemoveFile deletes path when it is empty of meaningful content. Under a dry
// run it reports the pending removal without deleting anything.
func RemoveFile(path string, opts Options) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if opts.DryRun {
		return true, nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return false, err
	}
	return true, nil
}

// RemoveDirIfEmpty deletes a directory only when it holds no entries, so an
// uninstall never discards files the host or the user still needs.
func RemoveDirIfEmpty(path string, opts Options) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if len(entries) != 0 {
		return false, nil
	}
	if opts.DryRun {
		return true, nil
	}
	if err := os.Remove(path); err != nil {
		return false, err
	}
	return true, nil
}
