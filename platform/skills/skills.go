// Package skills installs Agent Skills into the skill directories the agent
// hosts read. It mirrors the platform package: a table names each host's
// directory, and the shared writer performs the merge.
//
// Only the SKILL.md file is managed. A skill that needs supporting files can
// still ship them, but this package will not claim or delete them.
package skills

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

// All is the selector that expands to every supported skill host.
const All = "all"

// Both is the selector that expands to the two general-purpose skill roots:
// Claude Code and the shared .agents directory.
const Both = "both"

// Skill is one Agent Skill. Name becomes the directory under the host's skills
// root, and Content is the complete SKILL.md body, frontmatter included.
type Skill struct {
	Name    string
	Content string
}

// Options carries the ownership marker and the install decisions that apply to
// every skill host.
type Options struct {
	// Marker is the ownership token embedded in a managed SKILL.md as
	// "<!-- <Marker>-managed: skill -->". Required.
	Marker string
	// Force overwrites a same-named skill that does not carry the marker.
	Force bool
	// DryRun reports pending writes and removals without touching the disk.
	DryRun bool
}

// Descriptor describes one host's Agent Skills directory.
type Descriptor struct {
	// Agent is the selector written by the user.
	Agent string
	// Label names the host in listings.
	Label string
	// Scopes are the scopes this host supports.
	Scopes []platform.Scope
	// Path resolves the skills directory for a target.
	Path func(platform.Target) (string, error)
}

var descriptors = []Descriptor{
	{
		Agent:  "claude",
		Label:  "Claude Code skills",
		Scopes: []platform.Scope{platform.ScopeProject, platform.ScopeGlobal},
		Path:   claudePath,
	},
	{
		Agent:  "agents",
		Label:  "Shared .agents skills",
		Scopes: []platform.Scope{platform.ScopeProject, platform.ScopeGlobal},
		Path:   agentsPath,
	},
	{
		Agent:  "devin",
		Label:  "Devin skills",
		Scopes: []platform.Scope{platform.ScopeProject, platform.ScopeGlobal},
		Path:   devinPath,
	},
	{
		Agent:  "grok",
		Label:  "Grok skills",
		Scopes: []platform.Scope{platform.ScopeProject, platform.ScopeGlobal},
		Path:   grokPath,
	},
	{
		Agent:  "copilot-cli",
		Label:  "GitHub Copilot CLI skills",
		Scopes: []platform.Scope{platform.ScopeProject, platform.ScopeGlobal},
		Path:   copilotCLIPath,
	},
	{
		Agent:  "hermes",
		Label:  "Hermes skills",
		Scopes: []platform.Scope{platform.ScopeProject, platform.ScopeGlobal},
		Path:   hermesPath,
	},
}

var aliases = map[string]string{
	"claude-code":        "claude",
	"claudecode":         "claude",
	"agent":              "agents",
	"agent-skills":       "agents",
	".agents":            "agents",
	"devin-cli":          "devin",
	"grok-build":         "grok",
	"grok-cli":           "grok",
	"copilot":            "copilot-cli",
	"github-copilot":     "copilot-cli",
	"github-copilot-cli": "copilot-cli",
	"hermes-agent":       "hermes",
}

// Adapters returns the skill descriptors keyed by canonical agent name.
func Adapters() map[string]Descriptor {
	out := make(map[string]Descriptor, len(descriptors))
	for _, descriptor := range descriptors {
		out[descriptor.Agent] = descriptor
	}
	return out
}

// Names returns the canonical agent names in listing order.
func Names() []string {
	names := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		names = append(names, descriptor.Agent)
	}
	return names
}

// Lookup resolves one canonical name.
func Lookup(name string) (Descriptor, bool) {
	canonical := normalize(name)
	if canonical == "" {
		return Descriptor{}, false
	}
	if mapped, ok := aliases[canonical]; ok {
		canonical = mapped
	}
	for _, descriptor := range descriptors {
		if descriptor.Agent == canonical {
			return descriptor, true
		}
	}
	return Descriptor{}, false
}

// ResolveList expands a comma-separated selector into canonical agent names.
// "both" selects Claude Code and the shared .agents root; "all" selects every
// supported host. Duplicates are removed in first-seen order.
func ResolveList(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("a skills agent is required (supported: " + strings.Join(append(Names(), Both, All), ", ") + ")")
	}

	var selected []string
	for _, part := range strings.Split(value, ",") {
		part = normalize(part)
		if part == "" {
			continue
		}
		switch part {
		case All:
			selected = append(selected, Names()...)
		case Both:
			selected = append(selected, "claude", "agents")
		default:
			descriptor, ok := Lookup(part)
			if !ok {
				return nil, fmt.Errorf("unsupported skills agent %q (supported: %s)", part, strings.Join(append(Names(), Both, All), ", "))
			}
			selected = append(selected, descriptor.Agent)
		}
	}
	if len(selected) == 0 {
		return nil, errors.New("a skills agent is required")
	}
	return unique(selected), nil
}

// SupportsScope reports whether a descriptor writes configuration for scope.
func (d Descriptor) SupportsScope(scope platform.Scope) bool {
	for _, candidate := range d.Scopes {
		if candidate == scope {
			return true
		}
	}
	return false
}

// Validate checks a skill list before an install touches any file.
func Validate(list []Skill, opts Options) error {
	return validate(list, opts, true)
}

// ValidateForRemove checks the parts of a skill list an uninstall needs. The
// content is irrelevant when the file is only being matched against the
// ownership marker.
func ValidateForRemove(list []Skill, opts Options) error {
	return validate(list, opts, false)
}

func validate(list []Skill, opts Options, requireContent bool) error {
	if strings.TrimSpace(opts.Marker) == "" {
		return errors.New("skill marker is required")
	}
	if len(list) == 0 {
		return errors.New("no skills are defined in the Spec")
	}
	seen := make(map[string]struct{}, len(list))
	for _, skill := range list {
		if err := validateName(skill.Name); err != nil {
			return err
		}
		if requireContent && strings.TrimSpace(skill.Content) == "" {
			return fmt.Errorf("skill %q has no content", skill.Name)
		}
		if _, ok := seen[skill.Name]; ok {
			return fmt.Errorf("skill %q is defined more than once", skill.Name)
		}
		seen[skill.Name] = struct{}{}
	}
	return nil
}

// InstallOne writes every skill under one host's skills directory. It returns
// the paths that changed; a skill already holding the same managed content is
// left untouched and omitted from the result.
func InstallOne(target platform.Target, agent string, list []Skill, opts Options) (paths []string, warnings []string, err error) {
	if err := Validate(list, opts); err != nil {
		return nil, nil, err
	}
	descriptor, ok := Lookup(agent)
	if !ok {
		return nil, nil, fmt.Errorf("unsupported skills agent %q", agent)
	}
	if !descriptor.SupportsScope(target.Scope) {
		return nil, nil, fmt.Errorf("%s has no %s-level skills directory", descriptor.Agent, target.Scope)
	}
	root, err := descriptor.Path(target)
	if err != nil {
		return nil, nil, err
	}

	type pendingWrite struct {
		path    string
		content string
	}
	pending := make([]pendingWrite, 0, len(list))

	// Preflight every skill before writing one, so an unmanaged conflict on a
	// later skill cannot leave an earlier skill already replaced.
	marker := markerLine(opts.Marker)
	for _, skill := range list {
		path := filepath.Join(root, skill.Name, "SKILL.md")
		existing, readErr := os.ReadFile(path)
		if readErr == nil && !opts.Force && !strings.Contains(string(existing), marker) {
			return nil, nil, fmt.Errorf("refusing to overwrite unmanaged skill %s; pass --force to replace it", path)
		}
		if readErr != nil && !os.IsNotExist(readErr) {
			return nil, nil, readErr
		}
		pending = append(pending, pendingWrite{path: path, content: MarkContent(skill.Content, opts.Marker)})
	}

	for _, item := range pending {
		changed, warning, writeErr := common.WriteText(item.path, []byte(item.content), common.Options{DryRun: opts.DryRun})
		if writeErr != nil {
			return paths, warnings, writeErr
		}
		if warning != "" {
			warnings = append(warnings, warning)
		}
		if changed {
			paths = append(paths, item.path)
		}
	}
	return paths, warnings, nil
}

// RemoveOne deletes the managed SKILL.md files for one host, preserving any
// file that does not carry the marker. Empty skill and skills directories are
// pruned as they are emptied.
func RemoveOne(target platform.Target, agent string, list []Skill, opts Options) (paths []string, warnings []string, err error) {
	if err := ValidateForRemove(list, opts); err != nil {
		return nil, nil, err
	}
	descriptor, ok := Lookup(agent)
	if !ok {
		return nil, nil, fmt.Errorf("unsupported skills agent %q", agent)
	}
	if !descriptor.SupportsScope(target.Scope) {
		return nil, nil, fmt.Errorf("%s has no %s-level skills directory", descriptor.Agent, target.Scope)
	}
	root, err := descriptor.Path(target)
	if err != nil {
		return nil, nil, err
	}
	marker := markerLine(opts.Marker)

	for _, skill := range list {
		dir := filepath.Join(root, skill.Name)
		path := filepath.Join(dir, "SKILL.md")
		existing, readErr := os.ReadFile(path)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				continue
			}
			return paths, warnings, readErr
		}
		if !strings.Contains(string(existing), marker) {
			warnings = append(warnings, fmt.Sprintf("preserved unmanaged skill %s", path))
			continue
		}
		removed, removeErr := common.RemoveFile(path, common.Options{DryRun: opts.DryRun})
		if removeErr != nil {
			return paths, warnings, removeErr
		}
		if removed {
			paths = append(paths, path)
		}
		if _, pruneErr := common.RemoveDirIfEmpty(dir, common.Options{DryRun: opts.DryRun}); pruneErr != nil {
			return paths, warnings, pruneErr
		}
	}
	if _, pruneErr := common.RemoveDirIfEmpty(root, common.Options{DryRun: opts.DryRun}); pruneErr != nil {
		return paths, warnings, pruneErr
	}
	return paths, warnings, nil
}

// MarkContent returns content with the ownership marker guaranteed to be
// present. The marker is inserted after YAML frontmatter when there is any, so
// the skill's metadata stays valid.
func MarkContent(content, marker string) string {
	line := markerLine(marker)
	if strings.Contains(content, line) {
		return ensureTrailingNewline(content)
	}

	if strings.HasPrefix(content, "---\r\n") {
		rest := content[len("---\r\n"):]
		if idx := strings.Index(rest, "\r\n---\r\n"); idx >= 0 {
			insertAt := len("---\r\n") + idx + len("\r\n---\r\n")
			body := strings.TrimPrefix(content[insertAt:], "\r\n")
			return ensureTrailingNewline(content[:insertAt] + "\r\n" + line + "\r\n\r\n" + body)
		}
		if strings.HasSuffix(rest, "\r\n---") {
			return ensureTrailingNewline(content + "\r\n\r\n" + line)
		}
	}
	if strings.HasPrefix(content, "---\n") {
		rest := content[len("---\n"):]
		if idx := strings.Index(rest, "\n---\n"); idx >= 0 {
			insertAt := len("---\n") + idx + len("\n---\n")
			body := strings.TrimPrefix(content[insertAt:], "\n")
			return ensureTrailingNewline(content[:insertAt] + "\n" + line + "\n\n" + body)
		}
		if strings.HasSuffix(rest, "\n---") {
			return ensureTrailingNewline(content + "\n\n" + line)
		}
	}
	return ensureTrailingNewline(line + "\n\n" + content)
}

func markerLine(marker string) string {
	return "<!-- " + strings.TrimSpace(marker) + "-managed: skill -->"
}

func ensureTrailingNewline(content string) string {
	if strings.HasSuffix(content, "\n") {
		return content
	}
	return content + "\n"
}

func validateName(name string) error {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return errors.New("skill name is required")
	case name == "." || name == "..":
		return fmt.Errorf("invalid skill name %q", name)
	case strings.ContainsAny(name, `/\`):
		return fmt.Errorf("invalid skill name %q: it must be one path segment", name)
	}
	return nil
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

func claudePath(target platform.Target) (string, error) {
	if target.Scope == platform.ScopeProject {
		return target.Rel(".claude", "skills"), nil
	}
	return filepath.Join(envRoot("CLAUDE_CONFIG_DIR", filepath.Join(target.Root, ".claude")), "skills"), nil
}

func agentsPath(target platform.Target) (string, error) {
	return target.Rel(".agents", "skills"), nil
}

func devinPath(target platform.Target) (string, error) {
	if target.Scope == platform.ScopeProject {
		return target.Rel(".devin", "skills"), nil
	}
	if runtime.GOOS == "windows" {
		base := strings.TrimSpace(os.Getenv("APPDATA"))
		if base == "" {
			base = filepath.Join(target.Root, "AppData", "Roaming")
		}
		return filepath.Join(base, "devin", "skills"), nil
	}
	return target.Rel(".devin", "skills"), nil
}

func grokPath(target platform.Target) (string, error) {
	if target.Scope == platform.ScopeProject {
		return target.Rel(".grok", "skills"), nil
	}
	return filepath.Join(envRoot("GROK_HOME", filepath.Join(target.Root, ".grok")), "skills"), nil
}

func copilotCLIPath(target platform.Target) (string, error) {
	if target.Scope == platform.ScopeProject {
		return target.Rel(".github", "skills"), nil
	}
	return filepath.Join(envRoot("COPILOT_HOME", filepath.Join(target.Root, ".copilot")), "skills"), nil
}

func hermesPath(target platform.Target) (string, error) {
	return target.Rel(".hermes", "skills"), nil
}

func envRoot(variable, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(variable)); value != "" {
		return value
	}
	return fallback
}
