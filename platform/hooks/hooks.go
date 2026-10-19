// Package hooks installs lifecycle hooks into the host harnesses that expose
// a machine-readable hook configuration.
//
// The caller describes hooks in portable lifecycle terms (session-start,
// pre-tool-use, and so on). Each descriptor maps those events onto the host's
// own vocabulary and writes the host's native shape. Hooks are owned by the
// marker embedded in their command, or by a generated id or name prefix for
// hosts whose schema has no command marker. A re-apply replaces only this
// tool's entries and leaves other tools' hooks in place.
package hooks

import (
	"errors"
	"fmt"
	"strings"

	"github.com/phillarmonic/mcpinstall/platform"
)

// All is the selector that expands to every supported hook host.
const All = "all"

// Event is a portable lifecycle event. A host descriptor maps it onto the
// vocabulary that host uses.
type Event string

const (
	// SessionStart fires when an agent session begins.
	SessionStart Event = "session-start"
	// UserPromptSubmit fires when the user submits a prompt.
	UserPromptSubmit Event = "user-prompt"
	// PreToolUse fires before a tool call.
	PreToolUse Event = "pre-tool-use"
	// PostToolUse fires after a successful tool call.
	PostToolUse Event = "post-tool-use"
	// PostToolUseFailure fires after a failed tool call.
	PostToolUseFailure Event = "post-tool-use-failure"
	// PreCompact fires before context compaction.
	PreCompact Event = "pre-compact"
	// PostCompact fires after context compaction.
	PostCompact Event = "post-compact"
	// Stop fires at the end of an agent turn.
	Stop Event = "stop"
	// SessionEnd fires when the session is torn down.
	SessionEnd Event = "session-end"
	// SubagentStart fires when a subagent begins.
	SubagentStart Event = "subagent-start"
	// SubagentStop fires when a subagent ends.
	SubagentStop Event = "subagent-stop"
)

// Hook is one managed lifecycle hook.
//
// Command is the executable or shell command. Args are appended for hosts
// that execute a command and argument vector directly; hosts that take a
// single shell string receive the quoted join of Command and Args. Matcher is
// the optional tool-name matcher; hosts without matcher support ignore it with
// a warning.
type Hook struct {
	Event          Event
	Matcher        string
	Command        string
	Args           []string
	ID             string
	TimeoutSeconds int
}

// Options carries the ownership marker and the run direction shared by every
// hook host.
type Options struct {
	// Marker is the ownership token. A shell-form hook command must contain
	// it; hosts with a generated id or name use it as that identifier's
	// prefix. Required.
	Marker string
	// DryRun reports pending changes without touching the disk.
	DryRun bool
}

// Shape is the configuration shape a host expects.
type Shape int

const (
	// NestedJSON writes {"hooks": {Event: [{matcher, hooks: [{type, command}]}]}}.
	NestedJSON Shape = iota
	// FlatJSON writes {"version": 1, "hooks": {Event: [{type, command, matcher}]}}.
	FlatJSON
	// AntigravityJSON writes {Group: {Event: [...]}} with nested tool events
	// and flat lifecycle events.
	AntigravityJSON
	// ZeroJSON writes {"enabled": true, "hooks": [{id, event, command, args}]}.
	ZeroJSON
	// ZCodeJSON writes the events block inside the root hooks object.
	ZCodeJSON
	// KimiTOML writes [[hooks]] entries into config.toml.
	KimiTOML
	// KiroV3JSON writes {"version": "v1", "hooks": [{name, trigger, action}]}.
	KiroV3JSON
)

// Descriptor describes one host's hook configuration.
type Descriptor struct {
	Agent  string
	Label  string
	Scopes []platform.Scope
	// Path resolves the configuration file. The marker is used by hosts that
	// own a file per tool, such as Grok and Copilot CLI.
	Path func(platform.Target, string) (string, error)
	// Shape selects the writer.
	Shape Shape
	// Events maps portable events onto the host's event names. An absent
	// mapping means the host has no equivalent and the hook is skipped.
	Events map[Event]string
	// RootWrapper is true when the event map lives under a "hooks" key.
	RootWrapper bool
	// NoMatcher omits the matcher field for hosts whose schema rejects it.
	NoMatcher bool
	// Version is written at the top level when non-zero and absent.
	Version int
	// AddSchemaVersion writes schema_version 1 when it is absent.
	AddSchemaVersion bool
	// ToolEvents marks the events that use the nested shape in
	// AntigravityJSON.
	ToolEvents map[Event]bool
	// DeleteWhenEmpty removes the file once the last managed hook is gone.
	DeleteWhenEmpty bool
}

// SupportsScope reports whether this descriptor writes the requested scope.
func (d Descriptor) SupportsScope(scope platform.Scope) bool {
	for _, candidate := range d.Scopes {
		if candidate == scope {
			return true
		}
	}
	return false
}

// Adapters returns the supported descriptors keyed by canonical agent name.
func Adapters() map[string]Descriptor {
	out := make(map[string]Descriptor, len(descriptors))
	for _, descriptor := range descriptors {
		out[descriptor.Agent] = descriptor
	}
	return out
}

// Names returns the canonical hook host names in listing order.
func Names() []string {
	names := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		names = append(names, descriptor.Agent)
	}
	return names
}

// Lookup resolves one canonical name or alias.
func Lookup(name string) (Descriptor, bool) {
	canonical := canonicalName(name)
	for _, descriptor := range descriptors {
		if descriptor.Agent == canonical {
			return descriptor, true
		}
	}
	return Descriptor{}, false
}

// ResolveList expands a comma-separated selector into canonical host names.
// "all" selects every supported host. An explicitly selected unsupported host
// returns the reason it cannot be installed.
func ResolveList(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("a hooks agent is required (supported: " + strings.Join(append(Names(), All), ", ") + ")")
	}

	var selected []string
	for _, part := range strings.Split(value, ",") {
		part = normalize(part)
		if part == "" {
			continue
		}
		if part == All {
			selected = append(selected, Names()...)
			continue
		}
		canonical := canonicalName(part)
		if _, ok := Lookup(canonical); ok {
			selected = append(selected, canonical)
			continue
		}
		if reason, ok := unsupported[canonical]; ok {
			return nil, fmt.Errorf("unsupported hooks host %q: %s", part, reason)
		}
		return nil, fmt.Errorf("unsupported hooks host %q (supported: %s)", part, strings.Join(append(Names(), All), ", "))
	}
	if len(selected) == 0 {
		return nil, errors.New("a hooks agent is required")
	}
	return unique(selected), nil
}

// Validate checks the portable parts of a hook list before an install.
func Validate(list []Hook, opts Options) error {
	return validate(list, opts, true)
}

// ValidateForRemove checks the parts an uninstall needs.
func ValidateForRemove(list []Hook, opts Options) error {
	return validate(list, opts, false)
}

func validate(list []Hook, opts Options, requireCommand bool) error {
	if err := ValidateMarker(opts.Marker); err != nil {
		return err
	}
	if len(list) == 0 {
		return errors.New("no hooks are defined in the Spec")
	}
	for _, hook := range list {
		if strings.TrimSpace(string(hook.Event)) == "" {
			return errors.New("hook event is required")
		}
		if !knownEvent(hook.Event) {
			return fmt.Errorf("unsupported hook event %q", hook.Event)
		}
		if requireCommand && strings.TrimSpace(hook.Command) == "" {
			return fmt.Errorf("hook for event %q has no command", hook.Event)
		}
	}
	return nil
}

func knownEvent(event Event) bool {
	switch event {
	case SessionStart, UserPromptSubmit, PreToolUse, PostToolUse, PostToolUseFailure,
		PreCompact, PostCompact, Stop, SessionEnd, SubagentStart, SubagentStop:
		return true
	default:
		return false
	}
}

// ValidateMarker rejects a marker that cannot safely double as a file name or
// JSON group name.
func ValidateMarker(marker string) error {
	marker = strings.TrimSpace(marker)
	switch {
	case marker == "":
		return errors.New("hook marker is required")
	case marker == "." || marker == ".." || strings.HasPrefix(marker, "."):
		return fmt.Errorf("invalid hook marker %q", marker)
	}
	for _, r := range marker {
		valid := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.'
		if !valid {
			return fmt.Errorf("invalid hook marker %q: use letters, digits, '.', '-' or '_'", marker)
		}
	}
	return nil
}

// InstallOne writes the managed hooks for one host.
func InstallOne(target platform.Target, agent string, list []Hook, opts Options) (paths []string, warnings []string, err error) {
	if err := Validate(list, opts); err != nil {
		return nil, nil, err
	}
	opts.Marker = strings.TrimSpace(opts.Marker)
	descriptor, ok := Lookup(agent)
	if !ok {
		return nil, nil, fmt.Errorf("unsupported hooks host %q", agent)
	}
	if !descriptor.SupportsScope(target.Scope) {
		return nil, nil, fmt.Errorf("%s has no %s-level hooks configuration", descriptor.Agent, target.Scope)
	}
	for _, hook := range list {
		if descriptor.generatedIdentity() {
			continue
		}
		if !strings.Contains(commandLine(hook), opts.Marker) {
			return nil, nil, fmt.Errorf("hook command for event %q must contain the marker %q", hook.Event, opts.Marker)
		}
	}
	path, err := descriptor.Path(target, opts.Marker)
	if err != nil {
		return nil, nil, err
	}
	return writeInstall(descriptor, path, list, opts)
}

// RemoveOne deletes the managed hooks for one host.
func RemoveOne(target platform.Target, agent string, list []Hook, opts Options) (paths []string, warnings []string, err error) {
	if err := ValidateForRemove(list, opts); err != nil {
		return nil, nil, err
	}
	opts.Marker = strings.TrimSpace(opts.Marker)
	descriptor, ok := Lookup(agent)
	if !ok {
		return nil, nil, fmt.Errorf("unsupported hooks host %q", agent)
	}
	if !descriptor.SupportsScope(target.Scope) {
		return nil, nil, fmt.Errorf("%s has no %s-level hooks configuration", descriptor.Agent, target.Scope)
	}
	path, err := descriptor.Path(target, opts.Marker)
	if err != nil {
		return nil, nil, err
	}
	return writeRemove(descriptor, path, list, opts)
}

// generatedIdentity reports whether ownership is carried by a generated id or
// name rather than by a marker in the command.
func (d Descriptor) generatedIdentity() bool {
	switch d.Shape {
	case ZeroJSON, ZCodeJSON, KiroV3JSON:
		return true
	default:
		return false
	}
}

func canonicalName(name string) string {
	name = normalize(name)
	if mapped, ok := aliases[name]; ok {
		return mapped
	}
	return name
}

func normalize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return strings.Trim(value, "-")
}

func unique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
