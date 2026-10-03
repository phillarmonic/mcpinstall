// Package install runs the host adapters for a command-line request: it
// resolves the scope root, invokes one adapter per requested host, and collects
// the outcomes. It holds no host knowledge of its own, so a new host is picked
// up simply by appearing in the catalog.
package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phillarmonic/mcpinstall"
	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/catalog"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

// Request describes one install or uninstall run.
type Request struct {
	// Spec names the MCP server being installed and carries its guidance tree.
	Spec mcpinstall.Spec
	// Hosts are catalog names, already resolved from user input.
	Hosts []string
	// Scope selects project or global configuration.
	Scope platform.Scope
	// Root overrides the directory the scope resolves to. An empty value uses
	// the working directory for project scope and the home directory for
	// global scope.
	Root string
	// Options carries the command name, force and dry-run decisions. The
	// server identity fields are filled from Spec unless explicitly set.
	Options common.Options
}

// Result is the outcome of running one host adapter.
//
// The fields are ordered strings-then-slices to keep the garbage collector's
// pointer scan tight, so do not regroup them for readability.
type Result struct {
	// Host is the catalog name of the adapter that ran.
	Host string `json:"host"`
	// Error is set when this host failed, leaving the other hosts unaffected.
	Error string `json:"error,omitempty"`
	// Paths are the configuration files this host created, changed or removed.
	// Under a dry run they are the files that would change.
	Paths []string `json:"paths,omitempty"`
	// Warnings describe anything preserved rather than rewritten.
	Warnings []string `json:"warnings,omitempty"`
}

// resolveOptions fills the server identity from the spec, letting an explicit
// request override the command or arguments.
func resolveOptions(req Request) common.Options {
	opts := common.Options{
		ServerName: req.Spec.Name,
		Command:    req.Spec.Command,
		Args:       req.Spec.Args,
		EntryName:  req.Options.EntryName,
		Force:      req.Options.Force,
		DryRun:     req.Options.DryRun,
	}
	if req.Options.Command != "" {
		opts.Command = req.Options.Command
	}
	if len(req.Options.Args) > 0 {
		opts.Args = req.Options.Args
	}
	return opts
}

// Run executes the request against every requested host. It returns one result
// per host even when some fail, so a single bad host does not hide the rest;
// the returned error reports that at least one host failed.
func Run(req Request, removing bool) ([]Result, error) {
	root, err := ResolveRoot(req.Scope, req.Root)
	if err != nil {
		return nil, err
	}
	target := platform.Target{Scope: req.Scope, Root: root}
	opts := resolveOptions(req)

	adapters := catalog.Adapters()
	results := make([]Result, 0, len(req.Hosts))
	failures := 0

	for _, host := range req.Hosts {
		adapter, ok := adapters[host]
		if !ok {
			results = append(results, Result{Host: host, Error: fmt.Sprintf("unsupported host %q", host)})
			failures++
			continue
		}
		result := runOne(adapter, target, opts, removing)
		if result.Error != "" {
			failures++
		}
		results = append(results, result)
	}

	if failures > 0 {
		return results, fmt.Errorf("%d of %d hosts failed", failures, len(req.Hosts))
	}
	return results, nil
}

// runOne invokes a single adapter and normalises its outcome.
func runOne(adapter platform.Adapter, target platform.Target, opts common.Options, removing bool) Result {
	result := Result{Host: adapter.Name()}

	var (
		paths    []string
		warnings []string
		err      error
	)
	if removing {
		paths, warnings, err = adapter.Remove(target, opts)
	} else {
		paths, warnings, err = adapter.Install(target, opts)
	}
	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.Paths = make([]string, 0, len(paths))
	for _, path := range paths {
		result.Paths = append(result.Paths, displayPath(target.Root, path))
	}
	result.Warnings = warnings
	return result
}

// ResolveRoot turns a scope into the root adapters resolve their paths against:
// the working directory for project configuration, the home directory for
// global configuration. An explicit override wins over both.
func ResolveRoot(scope platform.Scope, override string) (string, error) {
	if override != "" {
		return override, nil
	}
	if scope == platform.ScopeGlobal {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("locate home directory: %w", err)
		}
		return home, nil
	}

	root, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("locate working directory: %w", err)
	}
	return root, nil
}

// displayPath renders a configuration path relative to the scope root, using a
// ~ prefix for global configuration, so output stays short and readable.
func displayPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	if rel == "." {
		return path
	}
	if root == homeDir() {
		sep := string(filepath.Separator)
		// A path outside the home directory has no "~" form. A redirected
		// XDG_CONFIG_HOME reaches one, and abbreviating it would print a path
		// like "~/../../tmp/x", which the reader cannot paste back.
		if rel == ".." || strings.HasPrefix(rel, ".."+sep) {
			return path
		}
		return "~" + sep + rel
	}
	return rel
}

// homeDir returns the home directory, or an empty string when it cannot be
// determined. It only affects how a path is printed.
func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

// UnchangedLabel describes a host that had nothing to do, in the direction it
// was asked to work.
//
// The two directions mean opposite things by "no change". An install whose
// entry is already in place has a configuration that is up to date, so it
// reports "already current". An uninstall that found no managed entry has
// nothing to take out, and calling that "already current" would assert a state
// the file does not hold: the entry it names is absent.
func UnchangedLabel(removing bool) string {
	if removing {
		return "not installed"
	}
	return "already current"
}

// Summary counts the hosts that changed something, for a closing line.
func Summary(results []Result, removing, dryRun bool) string {
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
