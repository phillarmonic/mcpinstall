// Package catalog resolves user-supplied host names onto the platform adapters
// that know how to configure them. It is the single place that decides which
// hosts exist, so adding a host means adding one adapter and one entry here.
package catalog

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/claude"
	"github.com/phillarmonic/mcpinstall/platform/codex"
	"github.com/phillarmonic/mcpinstall/platform/copilot"
	"github.com/phillarmonic/mcpinstall/platform/copilotjetbrains"
	"github.com/phillarmonic/mcpinstall/platform/cursor"
	"github.com/phillarmonic/mcpinstall/platform/hosts"
	"github.com/phillarmonic/mcpinstall/platform/opencode"
)

// Host names accepted on the command line.
const (
	Claude           = "claude"
	Cursor           = "cursor"
	Copilot          = "copilot"
	CopilotJetBrains = "copilot-jetbrains"
	Codex            = "codex"
	Opencode         = "opencode"
	OpenCode2        = "opencode2"
	ClaudeDesktop    = "claude-desktop"
	Gemini           = "gemini"
	OpenClaw         = "openclaw"
	OMP              = "omp"
	Antigravity      = "antigravity"
	Zero             = "zero"
	ZCode            = "zcode"
	Devin            = "devin"
	Grok             = "grok"
	Kimi             = "kimi"
	Kiro             = "kiro"
	CommandCode      = "command-code"
	IBMBob           = "ibm-bob"
	CopilotCLI       = "copilot-cli"
	Droid            = "droid"
	Swival           = "swival"
	Zed              = "zed"
	Muse             = "muse"
	PrimeAgent       = "prime-agent"
	DSH              = "dsh"
	Pi               = "pi"
)

// All is the meta host name that selects every supported host at once. It is
// deliberately not an adapter: install.Run looks adapters up by name, and one
// entry standing for every host would not fit that contract. It is expanded
// during resolution instead.
const All = "all"

// Adapters returns every known host by catalog name. A host whose status is
// unsupported is still present so an explicit selection can explain itself;
// Names filters it out of "all".
func Adapters() map[string]platform.Adapter {
	adapters := map[string]platform.Adapter{
		Claude:           claude.Adapter{},
		Cursor:           cursor.Adapter{},
		Copilot:          copilot.Adapter{},
		CopilotJetBrains: copilotjetbrains.Adapter{},
		Codex:            codex.Adapter{},
		Opencode:         opencode.Adapter{},
	}
	for name, adapter := range hosts.All() {
		adapters[name] = adapter
	}
	return adapters
}

// Names returns the concrete, supported host names in sorted order, one per
// adapter. It is what the "all" meta name expands to. An adapter that only
// carries guidance (dsh, pi) is still selectable by name but is not swept into
// "all", because it has no configuration file to write.
func Names() []string {
	adapters := Adapters()
	names := make([]string, 0, len(adapters))
	for name, adapter := range adapters {
		if adapter.Status().Supported {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// Selectors returns every value the --host flag accepts, in sorted order: the
// concrete host names plus the "all" meta name, so help and usage text can
// advertise the meta name beside the hosts it covers.
func Selectors() []string {
	names := make([]string, 0, len(Adapters())+1)
	names = append(names, Names()...)
	names = append(names, All)
	sort.Strings(names)
	return names
}

// Statuses returns each adapter's capabilities keyed by catalog name.
func Statuses() map[string]platform.AdapterStatus {
	statuses := make(map[string]platform.AdapterStatus)
	for name, adapter := range Adapters() {
		statuses[name] = adapter.Status()
	}
	return statuses
}

// ResolveList resolves a comma-separated list of host names into the catalog
// names to act on, preserving the caller's order and dropping duplicates. The
// "all" meta name expands to every supported host where it appears, so
// "claude,all" and "all,cursor" both do what they read like.
//
// An empty entry is rejected so a typo such as "cursor,,codex" cannot be
// mistaken for a valid request.
func ResolveList(value, fallback string) ([]string, error) {
	if !strings.Contains(value, ",") {
		return resolveEntry(value, fallback)
	}

	seen := make(map[string]bool)
	resolved := make([]string, 0)
	for entry := range strings.SplitSeq(value, ",") {
		if strings.TrimSpace(entry) == "" {
			return nil, errors.New("host list contains an empty value")
		}
		names, err := resolveEntry(entry, "")
		if err != nil {
			return nil, err
		}
		for _, name := range names {
			if !seen[name] {
				seen[name] = true
				resolved = append(resolved, name)
			}
		}
	}
	return resolved, nil
}

// resolveEntry maps one host entry onto the catalog names it selects. Every name
// selects itself, while the meta name "all" selects every supported host. An
// empty entry falls back to fallback so callers can supply a default.
func resolveEntry(entry, fallback string) ([]string, error) {
	name := normalize(entry)
	if name == "" {
		name = normalize(fallback)
	}
	if name == All {
		return Names(), nil
	}
	if _, ok := Adapters()[name]; !ok {
		return nil, fmt.Errorf("unsupported host %q (supported: %s, or %s)", name, strings.Join(Names(), ", "), All)
	}
	return []string{name}, nil
}

// normalize folds the spelling variations a user might type onto a catalog
// name, so "Copilot JetBrains", "chatgpt", and "openai" all resolve alike.
func normalize(name string) string {
	fields := strings.Fields(strings.TrimSpace(strings.ToLower(name)))
	if len(fields) == 0 {
		return ""
	}
	folded := strings.ReplaceAll(strings.Join(fields, "-"), "_", "-")
	switch folded {
	case "openai", "chatgpt", "chat-gpt":
		return Codex
	case "claude-code":
		return Claude
	case "vscode", "vscode-copilot", "github-copilot":
		return Copilot
	case "github-copilot-cli":
		return CopilotCLI
	case "open-code":
		return Opencode
	case "opencode-v2", "open-code2":
		return OpenCode2
	case "gemini-cli":
		return Gemini
	case "antigravity-cli", "agy":
		return Antigravity
	case "kimi-code":
		return Kimi
	case "kiro-cli":
		return Kiro
	case "commandcode", "cmdc", "cmd":
		return CommandCode
	case "bob":
		return IBMBob
	case "factory-droid":
		return Droid
	case "oh-my-pi":
		return OMP
	case "muse-code", "musecode":
		return Muse
	case "primeagent":
		return PrimeAgent
	case "z-code":
		return ZCode
	case "open-claw":
		return OpenClaw
	case "deepseek-harness":
		return DSH
	default:
		return folded
	}
}
