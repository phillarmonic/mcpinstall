// Package mcpinstall registers a stdio MCP server with the configuration files
// of the agent hosts that support one (Claude Code, Cursor, Copilot in VS Code
// and JetBrains, Codex, and OpenCode), and writes the matching agent guidance.
//
// A caller supplies a Spec naming the server and pointing at its embedded
// guidance. Everything host-specific lives here, so each MCP project does not
// grow its own copy of the same adapters, merge logic, and CLI.
package mcpinstall

import "io/fs"

// Spec describes one MCP server for installation. It is the only input a host
// caller supplies; the mechanics of merging into each host's configuration are
// implemented by this module.
type Spec struct {
	// Name is the server key written into host configuration files, and the
	// key uninstall removes. Required.
	Name string
	// Command is the executable hosts launch (for example "backstage" or
	// "aekf-mcp"). Required.
	Command string
	// Args are fixed arguments stored alongside the command. Usually empty.
	Args []string
	// Program is the CLI name shown in usage text, such as "backstage".
	Program string
	// Marker prefixes the managed guidance section and names the Cursor rule:
	// "<Marker>-guidance" in the HTML comment markers and "<Marker>.mdc" for
	// the user-level Cursor rule. Required.
	Marker string
	// CursorRuleDescription is the alwaysApply frontmatter description for the
	// user-level Cursor rule. A sensible default is used when empty.
	CursorRuleDescription string
	// LegacyNames are former server keys uninstall-legacy removes, along with
	// guidance sections marked with "<name>-guidance". Empty disables legacy
	// cleanup.
	LegacyNames []string
	// Guidance is the embedded guidance tree. Files are read from GuidanceDir.
	// It may be nil when only MCP registration is used.
	Guidance fs.FS
	// GuidanceDir is the directory inside Guidance holding the harness files.
	// Empty means "guidance".
	GuidanceDir string
}

// SourceDir returns the directory inside Guidance holding the harness files.
func (s Spec) SourceDir() string {
	if s.GuidanceDir == "" {
		return "guidance"
	}
	return s.GuidanceDir
}

// CursorRule returns the filename of the user-level Cursor rule.
func (s Spec) CursorRule() string {
	return s.Marker + ".mdc"
}

// CursorDescription returns the Cursor rule description or a generic default.
func (s Spec) CursorDescription() string {
	if s.CursorRuleDescription != "" {
		return s.CursorRuleDescription
	}
	return "Prefer " + s.Program + " MCP tools over shell and IDE equivalents."
}
