package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillarmonic/mcpinstall/platform"
	"github.com/phillarmonic/mcpinstall/platform/catalog"
	"github.com/phillarmonic/mcpinstall/platform/common"
)

// request builds a run rooted at a temporary directory so tests never touch the
// working directory or the real home directory.
func request(root string, hosts ...string) Request {
	return Request{
		Spec:    testSpec,
		Hosts:   hosts,
		Scope:   platform.ScopeProject,
		Root:    root,
		Options: common.Options{},
	}
}

func TestRunInstallsEveryRequestedHost(t *testing.T) {
	root := t.TempDir()

	results, err := Run(request(root, catalog.Claude, catalog.Cursor), false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	for _, relative := range []string{".mcp.json", filepath.Join(".cursor", "mcp.json")} {
		if _, err := os.Stat(filepath.Join(root, relative)); err != nil {
			t.Fatalf("expected %s to exist: %v", relative, err)
		}
	}
}

func TestRunReportsPathsRelativeToRoot(t *testing.T) {
	root := t.TempDir()

	results, err := Run(request(root, catalog.Claude), false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(results[0].Paths) != 1 || results[0].Paths[0] != ".mcp.json" {
		t.Fatalf("paths = %v, want [.mcp.json]", results[0].Paths)
	}
}

func TestRunIsIdempotent(t *testing.T) {
	root := t.TempDir()

	if _, err := Run(request(root, catalog.Claude), false); err != nil {
		t.Fatalf("first run: %v", err)
	}
	results, err := Run(request(root, catalog.Claude), false)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if len(results[0].Paths) != 0 {
		t.Fatalf("a repeat run reported changes: %v", results[0].Paths)
	}
}

func TestRunRemoves(t *testing.T) {
	root := t.TempDir()

	if _, err := Run(request(root, catalog.Claude), false); err != nil {
		t.Fatalf("install: %v", err)
	}
	results, err := Run(request(root, catalog.Claude), true)
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if len(results[0].Paths) != 1 {
		t.Fatalf("paths = %v, want the removed file", results[0].Paths)
	}
	if _, err := os.Stat(filepath.Join(root, ".mcp.json")); !os.IsNotExist(err) {
		t.Fatal("configuration file survived uninstall")
	}
}

func TestRunKeepsGoingWhenOneHostFails(t *testing.T) {
	root := t.TempDir()

	results, err := Run(request(root, catalog.Claude, "not-a-host"), false)
	if err == nil {
		t.Fatal("expected an error when a host is unsupported")
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want one per host", len(results))
	}
	if results[1].Error == "" {
		t.Fatal("the unsupported host should carry an error")
	}
	// The healthy host must still have been installed.
	if _, statErr := os.Stat(filepath.Join(root, ".mcp.json")); statErr != nil {
		t.Fatalf("a failing host blocked a healthy one: %v", statErr)
	}
}

func TestRunReportsAUserLevelOnlyHostWithoutBlockingTheRest(t *testing.T) {
	root := t.TempDir()

	results, err := Run(request(root, catalog.CopilotJetBrains, catalog.Claude), false)
	if err != nil {
		t.Fatalf("a user-level-only host should be skipped, not fail the run: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want one per host", len(results))
	}
	if len(results[0].Warnings) == 0 || !strings.Contains(results[0].Warnings[0], "--scope global") {
		t.Fatalf("the skip does not say what to do instead: %q", results[0].Warnings)
	}
	// The healthy host must still have been installed.
	if _, statErr := os.Stat(filepath.Join(root, ".mcp.json")); statErr != nil {
		t.Fatalf("a failing host blocked a healthy one: %v", statErr)
	}
}

func TestRunExplainsAHostWithoutAWriter(t *testing.T) {
	root := t.TempDir()

	results, err := Run(request(root, catalog.DSH), false)
	if err == nil {
		t.Fatal("expected an error for a host with no writer")
	}
	if len(results) != 1 || !strings.Contains(results[0].Error, "DeepSeek Harness") {
		t.Fatalf("results = %#v, want the host's guidance", results)
	}
}

func TestRunDryRunWritesNothing(t *testing.T) {
	root := t.TempDir()
	req := request(root, catalog.Claude, catalog.Cursor)
	req.Options.DryRun = true

	results, err := Run(req, false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, result := range results {
		if len(result.Paths) == 0 {
			t.Fatalf("%s did not report a pending change", result.Host)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("dry run wrote %d entries", len(entries))
	}
}

func TestRunDispatchesToTheNamedAdapter(t *testing.T) {
	root := t.TempDir()

	results, err := Run(request(root, catalog.Copilot), false)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if results[0].Host != catalog.Copilot {
		t.Fatalf("host = %q, want %q", results[0].Host, catalog.Copilot)
	}
	if _, err := os.Stat(filepath.Join(root, ".vscode", "mcp.json")); err != nil {
		t.Fatalf("copilot adapter did not write its config: %v", err)
	}
}

func TestResolveOptionsUsesTheSpecToolModeUnlessOverridden(t *testing.T) {
	req := request(t.TempDir(), catalog.Claude)
	req.Spec.ToolMode = common.ToolModeNative

	if got := resolveOptions(req).ToolMode; got != common.ToolModeNative {
		t.Fatalf("ToolMode = %q, want the Spec value %q", got, common.ToolModeNative)
	}

	req.Options.ToolMode = common.ToolModeCodeMode
	if got := resolveOptions(req).ToolMode; got != common.ToolModeCodeMode {
		t.Fatalf("ToolMode = %q, want the request override %q", got, common.ToolModeCodeMode)
	}
}

func TestResolveRootHonoursOverride(t *testing.T) {
	root := t.TempDir()

	for _, scope := range []platform.Scope{platform.ScopeProject, platform.ScopeGlobal} {
		got, err := ResolveRoot(scope, root)
		if err != nil {
			t.Fatalf("ResolveRoot(%s): %v", scope, err)
		}
		if got != root {
			t.Fatalf("ResolveRoot(%s) = %q, want the override %q", scope, got, root)
		}
	}
}

func TestDisplayPathMarksHomeRelativePaths(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory available")
	}

	got := displayPath(home, filepath.Join(home, ".cursor", "mcp.json"))
	if got != "~"+string(filepath.Separator)+filepath.Join(".cursor", "mcp.json") {
		t.Fatalf("displayPath = %q", got)
	}
}

func TestDisplayPathKeepsPathsOutsideHomeWhole(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory available")
	}

	// A redirected configuration directory, which XDG_CONFIG_HOME allows, sits
	// outside the home directory and so has no "~" form.
	outside := filepath.Join(t.TempDir(), "github-copilot", "intellij", "mcp.json")
	if rel, err := filepath.Rel(home, outside); err == nil && !strings.HasPrefix(rel, "..") {
		t.Skip("temporary directory is inside the home directory")
	}

	if got := displayPath(home, outside); got != outside {
		t.Fatalf("displayPath = %q, want the whole path %q", got, outside)
	}
}

func TestSummaryCountsOutcomes(t *testing.T) {
	results := []Result{
		{Host: "claude", Paths: []string{".mcp.json"}},
		{Host: "cursor"},
		{Host: "codex", Error: "boom"},
	}

	got := Summary(results, false, false)
	want := "1 configured, 1 already current, 1 failed"
	if got != want {
		t.Fatalf("Summary = %q, want %q", got, want)
	}

	if got := Summary(results, false, true); got != "1 would change, 1 already current, 1 failed" {
		t.Fatalf("dry-run Summary = %q", got)
	}
	// An uninstall that changed nothing has nothing to remove, not a
	// configuration that is already right, so it must not reuse the install
	// wording.
	if got := Summary(results, true, false); got != "1 removed, 1 not installed, 1 failed" {
		t.Fatalf("uninstall Summary = %q", got)
	}
}

func TestUnchangedLabelFollowsDirection(t *testing.T) {
	if got := UnchangedLabel(false); got != "already current" {
		t.Fatalf("install label = %q", got)
	}
	if got := UnchangedLabel(true); got != "not installed" {
		t.Fatalf("uninstall label = %q", got)
	}
}
