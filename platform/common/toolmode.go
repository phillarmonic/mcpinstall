package common

import (
	"fmt"
	"strings"
)

// Tool surfaces a host can present for an MCP server.
//
// Code Mode groups a server's tools behind one execution tool, which keeps a
// large server out of the provider's tool list. Native puts every tool on the
// provider's own list, where hosts that route or batch tools directly can use
// them. A host without the distinction ignores the request.
const (
	// ToolModeCodeMode is the host default: tools are grouped by server.
	ToolModeCodeMode = "code-mode"
	// ToolModeNative exposes each tool on the provider's native tool list.
	ToolModeNative = "native"
)

// NativeTools reports whether opts asks for the direct tool surface.
func NativeTools(opts Options) bool {
	return normalizeToolMode(opts.ToolMode) == ToolModeNative
}

// ValidateToolMode rejects a mode the platform does not know. An empty value
// is valid and means "use the host default".
func ValidateToolMode(mode string) error {
	switch normalizeToolMode(mode) {
	case "", ToolModeCodeMode, ToolModeNative:
		return nil
	default:
		return fmt.Errorf("unsupported tool mode %q (supported: %s, %s)", mode, ToolModeCodeMode, ToolModeNative)
	}
}

// normalizeToolMode lowercases and trims a mode, accepting the spellings a
// user is likely to type for OpenCode's codemode key.
func normalizeToolMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "codemode", "code_mode":
		return ToolModeCodeMode
	case "native-tools", "native_tools":
		return ToolModeNative
	default:
		return strings.ToLower(strings.TrimSpace(mode))
	}
}
