package hosts

import (
	"errors"
	"sort"
	"strings"

	"github.com/phillarmonic/mcpinstall/platform/common"
)

// errZCodeNeedsHTTP marks the one host that cannot express a stdio entry. The
// descriptor treats it as a skip with a warning, not a failure.
var errZCodeNeedsHTTP = errors.New("ZCode keeps only HTTP entries (type/url/headers); configure a URL for this host")

// entryFunc builds one host's server entry from the shared options.
type entryFunc func(common.Options) (map[string]any, error)

// plainStdio is the command/args shape most hosts accept without a
// transport discriminator.
func plainStdio(opts common.Options) (map[string]any, error) {
	return common.ManagedServer(opts), nil
}

// typedStdio adds the transport discriminator a typed host requires.
func typedStdio(kind string) entryFunc {
	return func(opts common.Options) (map[string]any, error) {
		return common.AddType(common.ManagedServer(opts), kind), nil
	}
}

// plainHTTP is the url/headers shape several hosts share.
func plainHTTP(opts common.Options) (map[string]any, error) {
	return common.ManagedHTTP(opts, "url"), nil
}

// typedHTTP adds the transport discriminator a typed host requires.
func typedHTTP(kind string) entryFunc {
	return func(opts common.Options) (map[string]any, error) {
		return common.AddType(common.ManagedHTTP(opts, "url"), kind), nil
	}
}

// chooseStdioHTTP picks a builder by transport.
func chooseStdioHTTP(stdio, http entryFunc) entryFunc {
	return func(opts common.Options) (map[string]any, error) {
		if common.IsHTTP(opts) {
			return http(opts)
		}
		return stdio(opts)
	}
}

// claudeDesktopEntry bridges HTTP through mcp-remote because Claude Desktop's
// own configuration file only starts stdio servers.
func claudeDesktopEntry(opts common.Options) (map[string]any, error) {
	if !common.IsHTTP(opts) {
		return common.ManagedServer(opts), nil
	}
	args := []any{"-y", "mcp-remote", opts.URL}
	for _, name := range sortedHeaderNames(opts.Headers) {
		args = append(args, "--header", name+": "+opts.Headers[name])
	}
	if opts.TokenEnvVar != "" && !hasHeader(opts, "Authorization") {
		args = append(args, "--header", "Authorization: Bearer ${"+opts.TokenEnvVar+"}")
	}
	return map[string]any{"command": "npx", "args": args}, nil
}

func geminiEntry(opts common.Options) (map[string]any, error) {
	if common.IsHTTP(opts) {
		entry := common.ManagedHTTP(opts, "httpUrl")
		entry["timeout"] = geminiTimeoutMs
		return entry, nil
	}
	entry := common.ManagedServer(opts)
	entry["timeout"] = geminiTimeoutMs
	return entry, nil
}

func antigravityEntry(opts common.Options) (map[string]any, error) {
	if common.IsHTTP(opts) {
		entry := common.ManagedHTTP(opts, "serverUrl")
		entry["timeout"] = geminiTimeoutMs
		return entry, nil
	}
	entry := common.ManagedServer(opts)
	entry["timeout"] = geminiTimeoutMs
	return entry, nil
}

func openclawEntry(opts common.Options) (map[string]any, error) {
	if common.IsHTTP(opts) {
		entry := common.ManagedHTTP(opts, "url")
		entry["transport"] = "streamable-http"
		return entry, nil
	}
	return common.ManagedServer(opts), nil
}

func ompEntry(opts common.Options) (map[string]any, error) {
	if common.IsHTTP(opts) {
		return common.AddEnabled(common.AddType(common.ManagedHTTP(opts, "url"), common.TransportHTTP)), nil
	}
	return common.AddEnabled(common.AddType(common.ManagedServer(opts), common.TransportStdio)), nil
}

func devinEntry(opts common.Options) (map[string]any, error) {
	if common.IsHTTP(opts) {
		entry := common.ManagedHTTP(opts, "url")
		entry["transport"] = common.TransportHTTP
		return entry, nil
	}
	return common.ManagedServer(opts), nil
}

func grokEntry(opts common.Options) (map[string]any, error) {
	if common.IsHTTP(opts) {
		return common.AddEnabled(common.ManagedHTTP(opts, "url")), nil
	}
	return common.AddEnabled(common.ManagedServer(opts)), nil
}

func commandCodeEntry(opts common.Options) (map[string]any, error) {
	if common.IsHTTP(opts) {
		entry := common.AddEnabled(common.ManagedHTTP(opts, "url"))
		entry["transport"] = common.TransportHTTP
		return entry, nil
	}
	entry := common.AddEnabled(common.ManagedServer(opts))
	entry["transport"] = common.TransportStdio
	return entry, nil
}

func ibmBobEntry(opts common.Options) (map[string]any, error) {
	if common.IsHTTP(opts) {
		entry := common.AddType(common.ManagedHTTP(opts, "url"), "streamable-http")
		// Bob's automatic negotiation probes server/discover first; pin it
		// to the documented legacy initialize path.
		entry["protocolNegotiation"] = "legacy"
		return entry, nil
	}
	return common.ManagedServer(opts), nil
}

func copilotCLIEntry(opts common.Options) (map[string]any, error) {
	var entry map[string]any
	if common.IsHTTP(opts) {
		entry = common.AddType(common.ManagedHTTP(opts, "url"), common.TransportHTTP)
	} else {
		entry = common.AddType(common.ManagedServer(opts), "local")
	}
	entry["tools"] = []any{"*"}
	if common.NativeTools(opts) {
		// Copilot CLI's tool search may defer a server's tools out of the
		// model's list. "never" keeps them always loaded, which is the direct
		// tool surface this mode asks for.
		entry["deferTools"] = "never"
	}
	return entry, nil
}

func droidEntry(opts common.Options) (map[string]any, error) {
	if !common.IsHTTP(opts) {
		return common.ManagedServer(opts), nil
	}
	entry := common.AddType(common.ManagedHTTP(opts, "url"), common.TransportHTTP)
	// Droid expands ${VAR} only in headers. Name the variable rather than
	// copying a bearer into the file when the caller supplied one, while
	// keeping any other headers the caller set.
	if opts.TokenEnvVar != "" && !hasHeader(opts, "Authorization") {
		headers, _ := entry["headers"].(map[string]any)
		if headers == nil {
			headers = map[string]any{}
		}
		headers["Authorization"] = "Bearer ${" + opts.TokenEnvVar + "}"
		entry["headers"] = headers
	}
	return entry, nil
}

func swivalEntry(opts common.Options) (map[string]any, error) {
	if common.IsHTTP(opts) {
		return common.AddType(common.ManagedHTTP(opts, "url"), common.TransportHTTP), nil
	}
	return common.AddType(common.ManagedServer(opts), common.TransportStdio), nil
}

func zedEntry(opts common.Options) (map[string]any, error) {
	if common.IsHTTP(opts) {
		return common.ManagedHTTP(opts, "url"), nil
	}
	return common.ManagedServer(opts), nil
}

func museEntry(opts common.Options) (map[string]any, error) {
	var entry map[string]any
	if common.IsHTTP(opts) {
		entry = common.AddType(common.ManagedHTTP(opts, "url"), common.TransportHTTP)
	} else {
		entry = common.AddType(common.ManagedServer(opts), common.TransportStdio)
	}
	// Muse defaults mode to required, which aborts the whole run when the
	// server cannot start. An MCP server must never gate the agent.
	entry["mode"] = "optional"
	return entry, nil
}

func opencode2Entry(opts common.Options) (map[string]any, error) {
	var entry map[string]any
	if common.IsHTTP(opts) {
		entry = common.AddType(common.ManagedHTTP(opts, "url"), "remote")
		// Without oauth:false the beta probes the endpoint for OAuth metadata
		// on every connect.
		entry["oauth"] = false
	} else {
		command := make([]any, 0, len(opts.Args)+1)
		command = append(command, opts.Command)
		for _, arg := range opts.Args {
			command = append(command, arg)
		}
		entry = map[string]any{"type": "local", "command": command}
	}
	if common.NativeTools(opts) {
		// OpenCode routes MCP tools through Code Mode by default, which groups
		// them under the execute tool. codemode:false keeps each tool on the
		// provider's native list, which tool-heavy servers prefer.
		entry["codemode"] = false
	}
	return entry, nil
}

func zcodeEntry(opts common.Options) (map[string]any, error) {
	if !common.IsHTTP(opts) {
		return nil, errZCodeNeedsHTTP
	}
	// ZCode's schema is strict: any key outside type/url/headers/enabled/
	// timeoutMs makes it drop the server silently.
	return common.AddType(common.ManagedHTTP(opts, "url"), common.TransportHTTP), nil
}

func primeAgentEntry(opts common.Options) (map[string]any, error) {
	if !common.IsHTTP(opts) {
		return common.ManagedServer(opts), nil
	}
	entry := common.AddType(map[string]any{"url": opts.URL}, common.TransportHTTP)
	if opts.TokenEnvVar != "" {
		// Prime Agent lists a user server only when it has an OAuth grant or
		// this key names a set variable. A literal header dispatches but
		// leaves the server out of the model's prompt.
		entry["bearerTokenEnvVar"] = opts.TokenEnvVar
	} else if headers := common.HeadersMap(opts); headers != nil {
		entry["headers"] = headers
	}
	return entry, nil
}

func sortedHeaderNames(headers map[string]string) []string {
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func hasHeader(opts common.Options, name string) bool {
	for existing := range opts.Headers {
		if strings.EqualFold(existing, name) {
			return true
		}
	}
	return false
}

// unsupported explains a host that has no configuration writer but is still
// worth naming, so an explicit selection produces guidance rather than
// "unsupported host".
func unsupported(reason string) entryFunc {
	return func(common.Options) (map[string]any, error) {
		return nil, errors.New(reason)
	}
}
