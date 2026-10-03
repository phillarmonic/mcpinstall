package opencode

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

const settingsLabel = "OpenCode MCP settings"

// serversKey is the server table OpenCode reads. It has only ever used "mcp",
// so the strict writers are used: an unrelated "mcpServers" object is left as
// the user wrote it rather than being renamed onto a live key.
const serversKey = "mcp"

// projectCandidates and globalCandidates list the configuration filenames
// OpenCode probes, in the order its own resolver checks them. It stops at the
// first candidate that exists and reads only that file, so the managed entry
// has to land in that file — writing any other name would update a
// configuration the host never loads.
//
// The orders differ by scope because OpenCode's own defaults do: a project
// favors opencode.json, the user-level directory favors opencode.jsonc.
var (
	projectCandidates = []string{
		"opencode.json",
		"opencode.jsonc",
		filepath.Join(".opencode", "opencode.json"),
		filepath.Join(".opencode", "opencode.jsonc"),
	}
	globalCandidates = []string{"opencode.jsonc", "opencode.json", "config.json"}
)

// mcpPath returns the OpenCode file holding MCP servers for the target scope.
// Both scopes have a configuration file, so neither is refused.
func mcpPath(target platform.Target) (string, error) {
	if target.Scope == platform.ScopeGlobal {
		return firstExisting(configDir(target.Root), globalCandidates)
	}
	return firstExisting(target.Root, projectCandidates)
}

// firstExisting returns the first candidate under dir that exists as a file, or
// the host's own default — the first candidate — when none does, which is what
// OpenCode's resolver returns for a configuration it has yet to create.
func firstExisting(dir string, candidates []string) (string, error) {
	for _, name := range candidates {
		candidate := filepath.Join(dir, name)
		info, err := os.Stat(candidate)
		switch {
		case err == nil && !info.IsDir():
			return candidate, nil
		case err == nil:
			// A directory where a file is expected is not a configuration
			// OpenCode can load; keep looking.
		case !errors.Is(err, os.ErrNotExist):
			return "", err
		}
	}
	return filepath.Join(dir, candidates[0]), nil
}

// configDir returns the directory OpenCode keeps its user-level configuration
// in. It honours XDG_CONFIG_HOME, which OpenCode itself resolves against.
func configDir(home string) string {
	return configDirFor(home, os.Getenv)
}

// configDirFor is configDir with the environment supplied, so both branches
// stay testable from whichever machine runs the suite.
func configDirFor(home string, lookupEnv func(string) string) string {
	if configHome := lookupEnv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, "opencode")
	}
	return filepath.Join(home, ".config", "opencode")
}

// serverEntry builds the local (stdio) MCP entry OpenCode expects. Its shape
// differs from every other JSON host: the command and its arguments share one
// array, and "type" is required so OpenCode can tell a local server from a
// remote one.
func serverEntry(opts common.Options) map[string]any {
	command := make([]any, 0, len(opts.Args)+1)
	command = append(command, opts.Command)
	for _, arg := range opts.Args {
		command = append(command, arg)
	}
	return map[string]any{
		"type":    "local",
		"command": command,
		"enabled": true,
	}
}

// upsertMCP merges the managed entry into the OpenCode configuration.
func upsertMCP(path string, opts common.Options) (bool, string, error) {
	return common.UpsertMCPServerInKeyOnly(
		path,
		settingsLabel,
		serversKey,
		opts.ServerName,
		serverEntry(opts),
		opts,
	)
}

func removeMCP(path string, opts common.Options) (bool, string, error) {
	return common.RemoveMCPServerInKeyOnly(path, settingsLabel, serversKey, common.EntryName(opts), opts)
}
