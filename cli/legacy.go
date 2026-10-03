package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/phillarmonic/mcpinstall"
	"github.com/phillarmonic/mcpinstall/install"
	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/catalog"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

type legacyScope int

const (
	legacyScopeProject legacyScope = iota
	legacyScopeGlobal
	legacyScopeBoth
)

func runUninstallLegacy(spec mcpinstall.Spec, args []string, stdout, stderr io.Writer) int {
	if len(spec.LegacyNames) == 0 {
		printf(stderr, "%s has no legacy server name configured\n", program(spec))
		return 2
	}

	var hosts, scopeName, root, format string
	var dryRun bool

	fs := flag.NewFlagSet(CommandUninstallLegacy, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&hosts, "host", catalog.All, "comma-separated hosts, or all (default: all)")
	fs.StringVar(&scopeName, "scope", "both", "project, global, or both (default: both)")
	fs.StringVar(&root, "root", "", "project directory, or the home override when scope is global")
	fs.StringVar(&format, "format", "text", "output format: text or json")
	fs.BoolVar(&dryRun, "dry-run", false, "report pending changes without writing")
	fs.Usage = func() {
		printf(stderr, "usage: %s %s [flags]\n\n", program(spec), CommandUninstallLegacy)
		printf(stderr, "Remove legacy MCP server entries and guidance files that still name them.\n")
		printf(stderr, "The current %s entry and guidance are left in place.\n\n", spec.Name)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if format != "text" && format != "json" {
		printf(stderr, "unsupported format %q (supported: text, json)\n", format)
		return 2
	}
	scope, err := parseLegacyScope(scopeName)
	if err != nil {
		printf(stderr, "%v\n", err)
		return 2
	}
	resolved, err := catalog.ResolveList(hosts, "")
	if err != nil {
		printf(stderr, "%v\n", err)
		return 2
	}

	type scopeReport struct {
		Scope   string           `json:"scope"`
		Root    string           `json:"root"`
		Results []install.Result `json:"results"`
	}
	reports := make([]scopeReport, 0, 2)
	failed := false

	for _, one := range scopesFor(scope) {
		override := root
		if scope == legacyScopeBoth && one == platform.ScopeGlobal {
			override = ""
		}
		resolvedRoot, rootErr := install.ResolveRoot(one, override)
		if rootErr != nil {
			printf(stderr, "%v\n", rootErr)
			return 1
		}

		results := make([]install.Result, 0, len(spec.LegacyNames)+1)
		for _, legacyName := range spec.LegacyNames {
			request := install.Request{
				Spec:  spec,
				Hosts: hostsForLegacy(one, resolved),
				Scope: one,
				Root:  override,
				Options: common.Options{
					EntryName: legacyName,
					DryRun:    dryRun,
				},
			}
			hostResults, runErr := install.Run(request, true)
			results = append(results, hostResults...)
			if runErr != nil {
				failed = true
				printf(stderr, "%s scope: %v\n", one, runErr)
			}
		}

		paths, guideErr := install.RemoveLegacyGuidance(spec, resolvedRoot, dryRun)
		switch {
		case guideErr != nil:
			results = append(results, install.Result{Host: "guidance", Error: guideErr.Error()})
			failed = true
			printf(stderr, "%s scope: %v\n", one, guideErr)
		default:
			results = append(results, install.Result{Host: "guidance", Paths: paths})
		}

		reports = append(reports, scopeReport{
			Scope:   string(one),
			Root:    resolvedRoot,
			Results: results,
		})
	}

	if format == "json" {
		payload := map[string]any{
			"action":  CommandUninstallLegacy,
			"dry_run": dryRun,
			"scopes":  reports,
		}
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(payload)
	} else {
		for _, report := range reports {
			writeText(stdout, platform.Scope(report.Scope), report.Root, install.Request{
				Options: common.Options{DryRun: dryRun},
			}, report.Results, true)
		}
	}
	if failed {
		return 1
	}
	return 0
}

func parseLegacyScope(name string) (legacyScope, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "both":
		return legacyScopeBoth, nil
	case string(platform.ScopeProject):
		return legacyScopeProject, nil
	case string(platform.ScopeGlobal):
		return legacyScopeGlobal, nil
	default:
		return 0, fmt.Errorf("unsupported scope %q (supported: project, global, both)", name)
	}
}

func scopesFor(scope legacyScope) []platform.Scope {
	switch scope {
	case legacyScopeProject:
		return []platform.Scope{platform.ScopeProject}
	case legacyScopeGlobal:
		return []platform.Scope{platform.ScopeGlobal}
	default:
		return []platform.Scope{platform.ScopeProject, platform.ScopeGlobal}
	}
}

// hostsForLegacy drops Copilot for JetBrains from a project pass that also
// names other hosts. That adapter has no project file, and a combined cleanup
// should still finish the hosts that do.
func hostsForLegacy(scope platform.Scope, hosts []string) []string {
	if scope != platform.ScopeProject || len(hosts) < 2 {
		return hosts
	}
	kept := make([]string, 0, len(hosts))
	for _, host := range hosts {
		if host == catalog.CopilotJetBrains {
			continue
		}
		kept = append(kept, host)
	}
	return kept
}
