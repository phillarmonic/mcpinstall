// Package cli implements the command-line surface for distributing MCP
// configuration. A host calls Run with its Spec; the command set is identical
// for every caller, so each project does not reimplement installation.
package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/phillarmonic/mcpinstall"
	"github.com/phillarmonic/mcpinstall/install"
	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/catalog"
	"github.com/phillarmonic/mcpinstall/platform/common"
	"github.com/phillarmonic/mcpinstall/platform/hooks"
	"github.com/phillarmonic/mcpinstall/platform/skills"
)

// Subcommand names.
const (
	CommandInstall           = "install"
	CommandUninstall         = "uninstall"
	CommandUninstallLegacy   = "uninstall-legacy"
	CommandInstallGuidance   = "install-guidance"
	CommandUninstallGuidance = "uninstall-guidance"
	CommandGlobalGuidance    = "global-guidance"
	CommandInstallSkills     = "install-skills"
	CommandUninstallSkills   = "uninstall-skills"
	CommandInstallHooks      = "install-hooks"
	CommandUninstallHooks    = "uninstall-hooks"
	CommandHosts             = "hosts"
	CommandVersion           = "version"
	CommandHelp              = "help"
)

// Version is the CLI version reported by the version subcommand. A caller sets
// it from its own build so the binary has one place to declare it.
var Version = "dev"

// options are the flags shared by install and uninstall. The booleans sit last
// so the string fields stay contiguous for the garbage collector's pointer
// scan.
type options struct {
	hosts       string
	scope       string
	root        string
	command     string
	transport   string
	url         string
	tokenEnvVar string
	toolMode    string
	format      string
	headers     headerFlags
	force       bool
	dryRun      bool
}

// headerFlags collects repeatable --header name=value flags.
type headerFlags map[string]string

func (h *headerFlags) String() string {
	if h == nil || len(*h) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(*h))
	for name, value := range *h {
		pairs = append(pairs, name+"="+value)
	}
	slices.Sort(pairs)
	return strings.Join(pairs, ",")
}

func (h *headerFlags) Set(value string) error {
	name, headerValue, ok := strings.Cut(value, "=")
	if !ok || strings.TrimSpace(name) == "" {
		return fmt.Errorf("header must be name=value")
	}
	if *h == nil {
		*h = headerFlags{}
	}
	(*h)[strings.TrimSpace(name)] = strings.TrimSpace(headerValue)
	return nil
}

// program returns the CLI name used in usage text.
func program(spec mcpinstall.Spec) string {
	if spec.Program != "" {
		return spec.Program
	}
	return "mcpinstall"
}

// Run executes a CLI invocation for spec and returns the process exit code.
// Unknown subcommands and invalid flags are usage errors rather than failures.
func Run(spec mcpinstall.Spec, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(spec, stderr)
		return 2
	}

	switch args[0] {
	case CommandInstall:
		return runApply(spec, args[1:], stdout, stderr, false)
	case CommandUninstall:
		return runApply(spec, args[1:], stdout, stderr, true)
	case CommandUninstallLegacy:
		return runUninstallLegacy(spec, args[1:], stdout, stderr)
	case CommandInstallGuidance:
		return runApplyGuidance(spec, args[1:], stdout, stderr, install.GuidanceRequest{})
	case CommandUninstallGuidance:
		return runApplyGuidance(spec, args[1:], stdout, stderr, install.GuidanceRequest{Removing: true})
	case CommandGlobalGuidance:
		return runGlobalGuidance(spec, args[1:], stdout, stderr)
	case CommandInstallSkills:
		return runApplySkills(spec, args[1:], stdout, stderr, false)
	case CommandUninstallSkills:
		return runApplySkills(spec, args[1:], stdout, stderr, true)
	case CommandInstallHooks:
		return runApplyHooks(spec, args[1:], stdout, stderr, false)
	case CommandUninstallHooks:
		return runApplyHooks(spec, args[1:], stdout, stderr, true)
	case CommandHosts:
		return runHosts(stdout)
	case CommandVersion:
		printf(stdout, "%s\n", Version)
		return 0
	case CommandHelp, "-h", "--help":
		printUsage(spec, stdout)
		return 0
	default:
		printf(stderr, "unknown command %q\n\n", args[0])
		printUsage(spec, stderr)
		return 2
	}
}

// runApply handles install and uninstall, which differ only in direction.
func runApply(spec mcpinstall.Spec, args []string, stdout, stderr io.Writer, removing bool) int {
	name := CommandInstall
	if removing {
		name = CommandUninstall
	}

	opts, err := parseApplyFlags(spec, name, args, stderr)
	if err != nil {
		return 2
	}

	hosts, err := catalog.ResolveList(opts.hosts, "")
	if err != nil {
		printf(stderr, "%v\n", err)
		return 2
	}
	scope, err := platform.ParseScope(opts.scope)
	if err != nil {
		printf(stderr, "%v\n", err)
		return 2
	}

	request := install.Request{
		Spec:  spec,
		Hosts: hosts,
		Scope: scope,
		Root:  opts.root,
		Options: common.Options{
			Command:     opts.command,
			Transport:   opts.transport,
			URL:         opts.url,
			TokenEnvVar: opts.tokenEnvVar,
			ToolMode:    opts.toolMode,
			Headers:     map[string]string(opts.headers),
			Force:       opts.force,
			DryRun:      opts.dryRun,
		},
	}

	root, err := install.ResolveRoot(request.Scope, request.Root)
	if err != nil {
		printf(stderr, "%v\n", err)
		return 1
	}

	results, runErr := install.Run(request, removing)

	if opts.format == "json" {
		writeJSON(stdout, scope, root, request, results, removing)
	} else {
		writeText(stdout, scope, root, request, results, removing)
	}

	if runErr != nil {
		printf(stderr, "%v\n", runErr)
		return 1
	}
	return 0
}

// parseApplyFlags parses the shared install/uninstall flag set.
func parseApplyFlags(spec mcpinstall.Spec, name string, args []string, stderr io.Writer) (options, error) {
	var opts options

	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&opts.hosts, "host", "", "comma-separated hosts to act on, or "+catalog.All+" (required)")
	fs.StringVar(&opts.scope, "scope", string(platform.ScopeProject), "configuration scope: project or global")
	fs.StringVar(&opts.root, "root", "", "directory the scope resolves against (default: cwd, or home for global scope)")
	fs.StringVar(&opts.command, "command", "", "executable hosts should launch (default \""+spec.Command+"\")")
	fs.StringVar(&opts.transport, "transport", "", "entry transport: stdio or http (default: http when --url is set)")
	fs.StringVar(&opts.url, "url", "", "remote MCP endpoint for http transport")
	fs.StringVar(&opts.tokenEnvVar, "token-env-var", "", "environment variable holding the bearer token for hosts that support it")
	fs.StringVar(&opts.toolMode, "tool-mode", "", "tool surface for hosts that support it: code-mode or native")
	fs.Var(&opts.headers, "header", "http header as name=value (repeatable)")
	fs.StringVar(&opts.format, "format", "text", "output format: text or json")
	fs.BoolVar(&opts.force, "force", false, "overwrite an existing managed entry")
	fs.BoolVar(&opts.dryRun, "dry-run", false, "report pending changes without writing")

	fs.Usage = func() {
		printf(stderr, "usage: %s %s --host <%s> [flags]\n\n", program(spec), name, strings.Join(catalog.Selectors(), "|"))
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	if strings.TrimSpace(opts.hosts) == "" {
		printf(stderr, "%s %s requires --host (supported: %s)\n\n", program(spec), name, strings.Join(catalog.Selectors(), ", "))
		fs.Usage()
		return options{}, errors.New("missing host")
	}
	if opts.format != "text" && opts.format != "json" {
		printf(stderr, "unsupported format %q (supported: text, json)\n", opts.format)
		return options{}, errors.New("unsupported format")
	}
	if err := common.ValidateToolMode(opts.toolMode); err != nil {
		printf(stderr, "%v\n", err)
		return options{}, err
	}
	switch strings.ToLower(strings.TrimSpace(opts.transport)) {
	case "", "auto", common.TransportStdio, common.TransportHTTP:
	default:
		printf(stderr, "unsupported transport %q (supported: stdio, http)\n", opts.transport)
		return options{}, errors.New("unsupported transport")
	}
	return opts, nil
}

// writeText renders a run as one line per host.
func writeText(w io.Writer, scope platform.Scope, root string, req install.Request, results []install.Result, removing bool) {
	if req.Options.DryRun {
		printf(w, "dry run: no files were written\n")
	}
	printf(w, "%s scope: %s\n\n", scope, root)

	var verb string
	switch {
	case req.Options.DryRun && removing:
		verb = "would remove"
	case req.Options.DryRun:
		verb = "would write"
	case removing:
		verb = "removed"
	default:
		verb = "wrote"
	}

	for _, result := range results {
		if result.Error != "" {
			printf(w, "%s: error: %s\n", result.Host, result.Error)
			continue
		}
		if len(result.Paths) == 0 {
			printf(w, "%s: %s\n", result.Host, install.UnchangedLabel(removing))
		}
		for _, path := range result.Paths {
			printf(w, "%s: %s %s\n", result.Host, verb, path)
		}
		for _, warning := range result.Warnings {
			printf(w, "%s: warning: %s\n", result.Host, warning)
		}
	}

	printf(w, "\n%s\n", install.Summary(results, removing, req.Options.DryRun))
}

// writeJSON renders a run as a machine-readable envelope.
func writeJSON(w io.Writer, scope platform.Scope, root string, req install.Request, results []install.Result, removing bool) {
	action := "install"
	if removing {
		action = "uninstall"
	}
	payload := map[string]any{
		"action":  action,
		"scope":   string(scope),
		"root":    root,
		"dry_run": req.Options.DryRun,
		"results": results,
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	// The envelope only holds strings, numbers and slices, so encoding cannot
	// fail for anything but a broken writer.
	_ = encoder.Encode(payload)
}

// runHosts lists the supported hosts and how each stores its configuration.
func runHosts(w io.Writer) int {
	statuses := catalog.Statuses()
	names := make([]string, 0, len(statuses))
	for name := range statuses {
		names = append(names, name)
	}
	slices.Sort(names)

	// Size the column to the longest name so the listing stays aligned as hosts
	// are added, rather than drifting into a hand-maintained width.
	width := len(catalog.All)
	for _, name := range names {
		width = max(width, len(name))
	}

	for _, name := range names {
		status := statuses[name]
		if !status.Supported {
			printf(w, "%-*s guidance only\n", width, name)
			continue
		}
		nativeTools := ""
		if status.Capabilities.NativeTools {
			nativeTools = "  native-tools"
		}
		printf(w, "%-*s %s  key=%s%s\n", width, name, status.Capabilities.Format, status.Capabilities.ServersKey, nativeTools)
	}
	printf(w, "%-*s %s\n", width, catalog.All, "every supported MCP host listed above")
	return 0
}

// runApplySkills handles install-skills and uninstall-skills.
func runApplySkills(spec mcpinstall.Spec, args []string, stdout, stderr io.Writer, removing bool) int {
	name := CommandInstallSkills
	if removing {
		name = CommandUninstallSkills
	}

	var (
		agent  string
		scope  string
		root   string
		format string
		force  bool
		dryRun bool
	)

	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&agent, "agent", "", "comma-separated skills hosts, or "+skills.All+" (required)")
	fs.StringVar(&scope, "scope", string(platform.ScopeProject), "skills scope: project or global")
	fs.StringVar(&root, "root", "", "directory the scope resolves against (default: cwd, or home for global scope)")
	fs.StringVar(&format, "format", "text", "output format: text or json")
	fs.BoolVar(&force, "force", false, "replace a same-named skill that is not managed")
	fs.BoolVar(&dryRun, "dry-run", false, "report pending changes without writing")

	fs.Usage = func() {
		printf(stderr, "usage: %s %s --agent <%s> [flags]\n\n", program(spec), name, strings.Join(append(skills.Names(), skills.Both, skills.All), "|"))
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(agent) == "" {
		printf(stderr, "%s %s requires --agent (supported: %s)\n\n", program(spec), name, strings.Join(append(skills.Names(), skills.Both, skills.All), ", "))
		fs.Usage()
		return 2
	}
	if format != "text" && format != "json" {
		printf(stderr, "unsupported format %q (supported: text, json)\n", format)
		return 2
	}

	agents, err := skills.ResolveList(agent)
	if err != nil {
		printf(stderr, "%v\n", err)
		return 2
	}
	scopeValue, err := platform.ParseScope(scope)
	if err != nil {
		printf(stderr, "%v\n", err)
		return 2
	}
	resolvedRoot, err := install.ResolveRoot(scopeValue, root)
	if err != nil {
		printf(stderr, "%v\n", err)
		return 1
	}

	results, runErr := install.RunSkills(install.SkillRequest{
		Spec:   spec,
		Agents: agents,
		Scope:  scopeValue,
		Root:   root,
		Force:  force,
		DryRun: dryRun,
	}, removing)

	if format == "json" {
		writeSkillsJSON(stdout, scopeValue, resolvedRoot, results, dryRun, removing)
	} else {
		writeSkillsText(stdout, scopeValue, resolvedRoot, results, removing, dryRun)
	}

	if runErr != nil {
		printf(stderr, "%v\n", runErr)
		return 1
	}
	return 0
}

func writeSkillsText(w io.Writer, scope platform.Scope, root string, results []install.SkillResult, removing, dryRun bool) {
	if dryRun {
		printf(w, "dry run: no files were written\n")
	}
	printf(w, "%s scope: %s\n\n", scope, root)

	verb := "wrote"
	switch {
	case dryRun && removing:
		verb = "would remove"
	case dryRun:
		verb = "would write"
	case removing:
		verb = "removed"
	}

	for _, result := range results {
		if result.Error != "" {
			printf(w, "%s: error: %s\n", result.Agent, result.Error)
			continue
		}
		if len(result.Paths) == 0 {
			printf(w, "%s: %s\n", result.Agent, install.UnchangedLabel(removing))
		}
		for _, path := range result.Paths {
			printf(w, "%s: %s %s\n", result.Agent, verb, path)
		}
		for _, warning := range result.Warnings {
			printf(w, "%s: warning: %s\n", result.Agent, warning)
		}
	}

	printf(w, "\n%s\n", install.SkillSummary(results, removing, dryRun))
}

func writeSkillsJSON(w io.Writer, scope platform.Scope, root string, results []install.SkillResult, dryRun, removing bool) {
	action := CommandInstallSkills
	if removing {
		action = CommandUninstallSkills
	}
	payload := map[string]any{
		"action":  action,
		"scope":   string(scope),
		"root":    root,
		"dry_run": dryRun,
		"results": results,
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(payload)
}

// runApplyHooks handles install-hooks and uninstall-hooks.
func runApplyHooks(spec mcpinstall.Spec, args []string, stdout, stderr io.Writer, removing bool) int {
	name := CommandInstallHooks
	if removing {
		name = CommandUninstallHooks
	}

	var (
		agent  string
		scope  string
		root   string
		format string
		dryRun bool
	)

	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&agent, "agent", "", "comma-separated hooks hosts, or "+hooks.All+" (required)")
	fs.StringVar(&scope, "scope", string(platform.ScopeGlobal), "hooks scope: project or global")
	fs.StringVar(&root, "root", "", "directory the scope resolves against (default: cwd, or home for global scope)")
	fs.StringVar(&format, "format", "text", "output format: text or json")
	fs.BoolVar(&dryRun, "dry-run", false, "report pending changes without writing")

	fs.Usage = func() {
		printf(stderr, "usage: %s %s --agent <%s> [flags]\n\n", program(spec), name, strings.Join(append(hooks.Names(), hooks.All), "|"))
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(agent) == "" {
		printf(stderr, "%s %s requires --agent (supported: %s)\n\n", program(spec), name, strings.Join(append(hooks.Names(), hooks.All), ", "))
		fs.Usage()
		return 2
	}
	if format != "text" && format != "json" {
		printf(stderr, "unsupported format %q (supported: text, json)\n", format)
		return 2
	}

	hosts, err := hooks.ResolveList(agent)
	if err != nil {
		printf(stderr, "%v\n", err)
		return 2
	}
	scopeValue, err := platform.ParseScope(scope)
	if err != nil {
		printf(stderr, "%v\n", err)
		return 2
	}
	resolvedRoot, err := install.ResolveRoot(scopeValue, root)
	if err != nil {
		printf(stderr, "%v\n", err)
		return 1
	}

	results, runErr := install.RunHooks(install.HookRequest{
		Spec:   spec,
		Hosts:  hosts,
		Scope:  scopeValue,
		Root:   root,
		DryRun: dryRun,
	}, removing)

	if format == "json" {
		writeHooksJSON(stdout, scopeValue, resolvedRoot, results, dryRun, removing)
	} else {
		writeHooksText(stdout, scopeValue, resolvedRoot, results, removing, dryRun)
	}

	if runErr != nil {
		printf(stderr, "%v\n", runErr)
		return 1
	}
	return 0
}

func writeHooksText(w io.Writer, scope platform.Scope, root string, results []install.HookResult, removing, dryRun bool) {
	if dryRun {
		printf(w, "dry run: no files were written\n")
	}
	printf(w, "%s scope: %s\n\n", scope, root)

	verb := "wrote"
	switch {
	case dryRun && removing:
		verb = "would remove"
	case dryRun:
		verb = "would write"
	case removing:
		verb = "removed"
	}

	for _, result := range results {
		if result.Error != "" {
			printf(w, "%s: error: %s\n", result.Host, result.Error)
			continue
		}
		if len(result.Paths) == 0 {
			printf(w, "%s: %s\n", result.Host, install.UnchangedLabel(removing))
		}
		for _, path := range result.Paths {
			printf(w, "%s: %s %s\n", result.Host, verb, path)
		}
		for _, warning := range result.Warnings {
			printf(w, "%s: warning: %s\n", result.Host, warning)
		}
	}

	printf(w, "\n%s\n", install.HookSummary(results, removing, dryRun))
}

func writeHooksJSON(w io.Writer, scope platform.Scope, root string, results []install.HookResult, dryRun, removing bool) {
	action := CommandInstallHooks
	if removing {
		action = CommandUninstallHooks
	}
	payload := map[string]any{
		"action":  action,
		"scope":   string(scope),
		"root":    root,
		"dry_run": dryRun,
		"results": results,
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(payload)
}

func runGlobalGuidance(spec mcpinstall.Spec, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printGlobalGuidanceUsage(spec, stderr)
		return 2
	}
	req := install.GuidanceRequest{Global: true}
	switch args[0] {
	case CommandInstall:
		return runApplyGuidance(spec, args[1:], stdout, stderr, req)
	case CommandUninstall:
		req.Removing = true
		return runApplyGuidance(spec, args[1:], stdout, stderr, req)
	case CommandHelp, "-h", "--help":
		printGlobalGuidanceUsage(spec, stdout)
		return 0
	default:
		printf(stderr, "unknown command %q\n\n", args[0])
		printGlobalGuidanceUsage(spec, stderr)
		return 2
	}
}

func printGlobalGuidanceUsage(spec mcpinstall.Spec, w io.Writer) {
	printf(w, `usage: %s global-guidance <install|uninstall> [flags]

Write user-level agent instructions under the home directory so a working
tree does not need project guidance files.

flags:
  --root     directory to treat as home (default: the user home directory)
  --harness  agent harness: copilot, cursor, codex, all (default: all)
  --force    overwrite existing guidance files
  --dry-run  report pending changes without writing
`, program(spec))
}

func guidanceCommandName(req install.GuidanceRequest) string {
	switch {
	case req.Global && req.Removing:
		return CommandGlobalGuidance + " " + CommandUninstall
	case req.Global:
		return CommandGlobalGuidance + " " + CommandInstall
	case req.Removing:
		return CommandUninstallGuidance
	default:
		return CommandInstallGuidance
	}
}

// runApplyGuidance handles project and user-level guidance install and uninstall.
func runApplyGuidance(spec mcpinstall.Spec, args []string, stdout, stderr io.Writer, req install.GuidanceRequest) int {
	name := guidanceCommandName(req)

	var root, harness string
	var force, dryRun bool

	rootHelp := "project root directory (default: current directory)"
	if req.Global {
		rootHelp = "directory to treat as home (default: the user home directory)"
	}

	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&root, "root", "", rootHelp)
	fs.StringVar(&harness, "harness", "all", "agent harness: copilot, cursor, codex, all (default: all)")
	fs.BoolVar(&force, "force", false, "overwrite existing guidance files")
	fs.BoolVar(&dryRun, "dry-run", false, "report pending changes without writing")

	fs.Usage = func() {
		if req.Global {
			printGlobalGuidanceUsage(spec, stderr)
			return
		}
		printf(stderr, "usage: %s %s [flags]\n\n", program(spec), name)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if root == "" {
		if req.Global {
			home, err := os.UserHomeDir()
			if err != nil {
				printf(stderr, "locate home directory: %v\n", err)
				return 1
			}
			root = home
		} else {
			root = "."
		}
	}

	req.Root = root
	req.Harness = harness
	req.Force = force
	req.DryRun = dryRun

	results, err := install.ApplyGuidance(spec, req)

	verb := "wrote"
	pastVerb := "written"
	if req.Removing {
		verb = "removed"
		pastVerb = "removed"
	}
	if req.DryRun {
		verb = "would write"
		if req.Removing {
			verb = "would remove"
		}
	}

	if len(results) == 0 && !req.Removing {
		printf(stdout, "guidance already current\n")
		return 0
	}
	if len(results) == 0 && req.Removing {
		printf(stdout, "guidance not installed\n")
		return 0
	}

	if req.DryRun {
		printf(stdout, "dry run: no files were written\n")
	}

	names := make([]string, 0, len(results))
	for harnessName := range results {
		names = append(names, harnessName)
	}
	slices.Sort(names)
	for _, harnessName := range names {
		printf(stdout, "%s: %s %s\n", harnessName, verb, install.FormatPath(root, results[harnessName]))
	}

	summary := pastVerb
	if req.DryRun {
		if req.Removing {
			summary = "would be removed"
		} else {
			summary = "would be written"
		}
	}
	printf(stdout, "\n%d harness guidance file(s) %s to %s\n", len(results), summary, root)

	if err != nil {
		printf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

// printUsage writes the top-level help.
func printUsage(spec mcpinstall.Spec, w io.Writer) {
	printf(w, `%[1]s: install the %[2]s MCP server, skills and hooks.

usage:
  %[1]s install              [flags]      register %[3]s with a host
  %[1]s uninstall            [flags]      remove %[3]s from a host
  %[1]s uninstall-legacy     [flags]      remove a previous server id and guidance
  %[1]s install-guidance     [flags]      install project agent guidance
  %[1]s uninstall-guidance   [flags]      remove project agent guidance
  %[1]s global-guidance install [flags]   install user-level agent guidance
  %[1]s global-guidance uninstall [flags] remove user-level agent guidance
  %[1]s install-skills       [flags]      install the Spec's Agent Skills
  %[1]s uninstall-skills     [flags]      remove the Spec's Agent Skills
  %[1]s install-hooks        [flags]      install the Spec's lifecycle hooks
  %[1]s uninstall-hooks      [flags]      remove the Spec's lifecycle hooks
  %[1]s hosts                             list supported MCP hosts
  %[1]s version                           print the version

flags for install and uninstall:
  --host     comma-separated hosts, or all (supported: %[4]s)
  --scope    project (default) or global
  --root     directory the scope resolves against
  --command      executable hosts should launch (default "%[3]s")
  --transport    stdio or http (default: http when --url is set)
  --url          remote MCP endpoint for http transport
  --header       http header as name=value (repeatable)
  --token-env-var environment variable holding the bearer token
  --tool-mode    code-mode (default) or native; native exposes each tool on
                 the provider's tool list where the host supports it
  --force    overwrite an existing managed entry
  --dry-run  report pending changes without writing
  --format   text (default) or json

flags for install-guidance, uninstall-guidance, and global-guidance:
  --root     project root, or home directory for global-guidance
  --harness  agent harness: copilot, cursor, codex, all (default: all)
  --force    overwrite existing guidance files
  --dry-run  report pending changes without writing

flags for install-skills and uninstall-skills:
  --agent    comma-separated skills hosts: claude, agents, devin, grok,
             copilot-cli, hermes, both, all (required)
  --scope    project (default) or global
  --root     directory the scope resolves against
  --force    replace a same-named skill that is not managed
  --dry-run  report pending changes without writing
  --format   text (default) or json

flags for install-hooks and uninstall-hooks:
  --agent    comma-separated hooks hosts, or all (required)
  --scope    global (default) or project (Claude Code only)
  --root     directory the scope resolves against
  --dry-run  report pending changes without writing
  --format   text (default) or json
`,
		program(spec), spec.Name, spec.Command, strings.Join(catalog.Selectors(), ", "))
}

// printf writes formatted output and discards the error. A CLI cannot recover
// from a broken stdout or stderr, and the exit code already reports the
// outcome, so threading these errors through every call site would add noise
// without changing behaviour.
func printf(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}
