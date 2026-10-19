package install

import (
	"fmt"
	"strings"

	"github.com/phillarmonic/mcpinstall"
	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/hooks"
)

// HookRequest describes one install-hooks or uninstall-hooks run.
type HookRequest struct {
	// Spec supplies the hooks and their ownership marker.
	Spec mcpinstall.Spec
	// Hosts are canonical hooks host names, already resolved from user input.
	Hosts []string
	// Scope selects project or global hook configuration.
	Scope platform.Scope
	// Root overrides the directory the scope resolves to.
	Root string
	// DryRun reports pending changes without writing.
	DryRun bool
}

// HookResult is the outcome of one hooks host.
type HookResult struct {
	Host     string   `json:"host"`
	Error    string   `json:"error,omitempty"`
	Paths    []string `json:"paths,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// RunHooks installs or removes the Spec's lifecycle hooks for every requested
// host. The returned error reports that at least one host failed; the results
// still describe every host.
func RunHooks(req HookRequest, removing bool) ([]HookResult, error) {
	root, err := ResolveRoot(req.Scope, req.Root)
	if err != nil {
		return nil, err
	}
	target := platform.Target{Scope: req.Scope, Root: root}

	opts := hooks.Options{
		Marker: req.Spec.HookOwnershipMarker(),
		DryRun: req.DryRun,
	}
	if removing {
		err = hooks.ValidateForRemove(req.Spec.Hooks, opts)
	} else {
		err = hooks.Validate(req.Spec.Hooks, opts)
	}
	if err != nil {
		return nil, err
	}

	adapters := hooks.Adapters()
	results := make([]HookResult, 0, len(req.Hosts))
	failures := 0
	firstFailure := ""

	for _, host := range req.Hosts {
		descriptor, ok := adapters[host]
		if !ok {
			message := fmt.Sprintf("unsupported hooks host %q", host)
			results = append(results, HookResult{Host: host, Error: message})
			if firstFailure == "" {
				firstFailure = message
			}
			failures++
			continue
		}
		if !descriptor.SupportsScope(req.Scope) {
			results = append(results, HookResult{
				Host:     host,
				Warnings: []string{fmt.Sprintf("skipped: no %s-level hooks configuration; pass --scope %s", req.Scope, otherScope(req.Scope))},
			})
			continue
		}

		var (
			paths    []string
			warnings []string
			runErr   error
		)
		if removing {
			paths, warnings, runErr = hooks.RemoveOne(target, host, req.Spec.Hooks, opts)
		} else {
			paths, warnings, runErr = hooks.InstallOne(target, host, req.Spec.Hooks, opts)
		}
		result := HookResult{Host: host}
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
		return results, fmt.Errorf("%d of %d hooks hosts failed: %s", failures, len(req.Hosts), firstFailure)
	}
	return results, nil
}

// HookSummary counts the hooks hosts that changed something.
func HookSummary(results []HookResult, removing, dryRun bool) string {
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
