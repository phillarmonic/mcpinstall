package install

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/mcpinstall"
	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/skills"
)

// SkillRequest describes one install-skills or uninstall-skills run.
type SkillRequest struct {
	// Spec supplies the skills and their ownership marker.
	Spec mcpinstall.Spec
	// Agents are canonical skills host names, already resolved from user input.
	Agents []string
	// Scope selects project or global skill directories.
	Scope platform.Scope
	// Root overrides the directory the scope resolves to.
	Root string
	// Force replaces a same-named skill that does not carry the marker.
	Force bool
	// DryRun reports pending changes without writing.
	DryRun bool
}

// SkillResult is the outcome of one skills host.
type SkillResult struct {
	Agent    string   `json:"agent"`
	Error    string   `json:"error,omitempty"`
	Paths    []string `json:"paths,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// RunSkills installs or removes the Spec's Agent Skills for every requested
// host. The returned error reports that at least one host failed; the results
// still describe every host.
func RunSkills(req SkillRequest, removing bool) ([]SkillResult, error) {
	root, err := ResolveRoot(req.Scope, req.Root)
	if err != nil {
		return nil, err
	}
	target := platform.Target{Scope: req.Scope, Root: root}

	opts := skills.Options{
		Marker: req.Spec.SkillOwnershipMarker(),
		Force:  req.Force,
		DryRun: req.DryRun,
	}
	if removing {
		err = skills.ValidateForRemove(req.Spec.Skills, opts)
	} else {
		err = skills.Validate(req.Spec.Skills, opts)
	}
	if err != nil {
		return nil, err
	}

	adapters := skills.Adapters()
	results := make([]SkillResult, 0, len(req.Agents))
	failures := 0
	firstFailure := ""

	for _, agent := range req.Agents {
		descriptor, ok := adapters[agent]
		if !ok {
			message := fmt.Sprintf("unsupported skills agent %q", agent)
			results = append(results, SkillResult{Agent: agent, Error: message})
			if firstFailure == "" {
				firstFailure = message
			}
			failures++
			continue
		}
		if !descriptor.SupportsScope(req.Scope) {
			results = append(results, SkillResult{
				Agent:    agent,
				Warnings: []string{fmt.Sprintf("skipped: no %s-level skills directory", req.Scope)},
			})
			continue
		}

		var (
			paths    []string
			warnings []string
			runErr   error
		)
		if removing {
			paths, warnings, runErr = skills.RemoveOne(target, agent, req.Spec.Skills, opts)
		} else {
			paths, warnings, runErr = skills.InstallOne(target, agent, req.Spec.Skills, opts)
		}
		result := SkillResult{Agent: agent}
		if runErr != nil {
			result.Error = runErr.Error()
			if firstFailure == "" {
				firstFailure = result.Error
			}
			failures++
		} else {
			result.Paths = make([]string, 0, len(paths))
			for _, path := range paths {
				result.Paths = append(result.Paths, displayPath(root, path))
			}
			result.Warnings = warnings
		}
		results = append(results, result)
	}

	if failures > 0 {
		return results, fmt.Errorf("%d of %d skills hosts failed: %s", failures, len(req.Agents), firstFailure)
	}
	return results, nil
}

// SkillSummary counts the skills hosts that changed something.
func SkillSummary(results []SkillResult, removing, dryRun bool) string {
	changed, unchanged, failed := 0, 0, 0
	for _, result := range results {
		switch {
		case result.Error != "":
			failed++
		case len(result.Paths) > 0:
			changed++
		default:
			unchanged++
		}
	}

	verb := "configured"
	switch {
	case dryRun:
		verb = "would change"
	case removing:
		verb = "removed"
	}
	parts := []string{fmt.Sprintf("%d %s", changed, verb)}
	if unchanged > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", unchanged, UnchangedLabel(removing)))
	}
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	return strings.Join(parts, ", ")
}
