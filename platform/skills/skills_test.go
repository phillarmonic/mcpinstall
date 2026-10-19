package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phillarmonic/mcpinstall/platform"
)

func clearSkillEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{"CLAUDE_CONFIG_DIR", "GROK_HOME", "COPILOT_HOME", "APPDATA"} {
		t.Setenv(name, "")
	}
}

func demoSkill() Skill {
	return Skill{
		Name:    "demo",
		Content: "---\nname: demo\ndescription: A demo skill\n---\n\n# Demo\n\nUse it.\n",
	}
}

func TestMarkContentInsertsAfterFrontmatter(t *testing.T) {
	got := MarkContent(demoSkill().Content, "backstage")
	wantMarker := "<!-- backstage-managed: skill -->"
	if !strings.Contains(got, wantMarker) {
		t.Fatalf("marker missing from %q", got)
	}
	if !strings.HasPrefix(got, "---\nname: demo\ndescription: A demo skill\n---\n\n"+wantMarker+"\n\n# Demo") {
		t.Fatalf("marker was not inserted after frontmatter:\n%s", got)
	}
}

func TestMarkContentPrependsWithoutFrontmatter(t *testing.T) {
	got := MarkContent("# Demo\n", "backstage")
	if !strings.HasPrefix(got, "<!-- backstage-managed: skill -->\n\n# Demo\n") {
		t.Fatalf("unexpected content:\n%s", got)
	}
}

func TestMarkContentKeepsCRLF(t *testing.T) {
	content := "---\r\nname: demo\r\n---\r\n\r\n# Demo\r\n"
	got := MarkContent(content, "backstage")
	if !strings.Contains(got, "---\r\n\r\n<!-- backstage-managed: skill -->\r\n\r\n# Demo") {
		t.Fatalf("unexpected content: %q", got)
	}
	if strings.Contains(strings.ReplaceAll(got, "\r\n", ""), "\n") {
		t.Fatalf("mixed line endings: %q", got)
	}
}

func TestMarkContentIsIdempotent(t *testing.T) {
	once := MarkContent(demoSkill().Content, "backstage")
	twice := MarkContent(once, "backstage")
	if once != twice {
		t.Fatalf("marker was added twice:\n%s\n---\n%s", once, twice)
	}
}

func TestInstallAndRemoveClaudeSkill(t *testing.T) {
	clearSkillEnv(t)
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeProject, Root: root}
	opts := Options{Marker: "backstage"}

	paths, warnings, err := InstallOne(target, "claude", []Skill{demoSkill()}, opts)
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}
	want := filepath.Join(root, ".claude", "skills", "demo", "SKILL.md")
	if len(paths) != 1 || paths[0] != want {
		t.Fatalf("paths = %v, want %s", paths, want)
	}
	data, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(data), "<!-- backstage-managed: skill -->") {
		t.Fatalf("installed file has no marker:\n%s", data)
	}

	again, _, err := InstallOne(target, "claude", []Skill{demoSkill()}, opts)
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("second install changed %v", again)
	}

	removed, _, err := RemoveOne(target, "claude", []Skill{{Name: "demo"}}, opts)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if len(removed) != 1 || removed[0] != want {
		t.Fatalf("removed = %v, want %s", removed, want)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "skills")); !os.IsNotExist(err) {
		t.Fatalf("empty skills tree was not pruned: %v", err)
	}
}

func TestInstallRefusesUnmanagedSkillAndForceReplacesIt(t *testing.T) {
	clearSkillEnv(t)
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeProject, Root: root}
	path := filepath.Join(root, ".claude", "skills", "demo", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# hand-written\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, _, err := InstallOne(target, "claude", []Skill{demoSkill()}, Options{Marker: "backstage"}); err == nil {
		t.Fatal("install overwrote an unmanaged skill without force")
	}
	if data, _ := os.ReadFile(path); string(data) != "# hand-written\n" {
		t.Fatalf("unmanaged file changed: %s", data)
	}

	if _, _, err := InstallOne(target, "claude", []Skill{demoSkill()}, Options{Marker: "backstage", Force: true}); err != nil {
		t.Fatalf("forced install: %v", err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "<!-- backstage-managed: skill -->") {
		t.Fatalf("forced install left no marker:\n%s", data)
	}
}

func TestRemovePreservesUnmanagedSkill(t *testing.T) {
	clearSkillEnv(t)
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeProject, Root: root}
	path := filepath.Join(root, ".claude", "skills", "demo", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("# hand-written\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	paths, warnings, err := RemoveOne(target, "claude", []Skill{{Name: "demo"}}, Options{Marker: "backstage"})
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if len(paths) != 0 {
		t.Fatalf("removed unmanaged path: %v", paths)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "preserved unmanaged") {
		t.Fatalf("warnings = %v", warnings)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("unmanaged file disappeared: %v", err)
	}
}

func TestGlobalSkillPathsHonorRelocations(t *testing.T) {
	clearSkillEnv(t)
	root := t.TempDir()
	target := platform.Target{Scope: platform.ScopeGlobal, Root: root}
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(root, "claude-home"))
	t.Setenv("GROK_HOME", filepath.Join(root, "grok-home"))
	t.Setenv("COPILOT_HOME", filepath.Join(root, "copilot-home"))

	claude, err := claudePath(target)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "claude-home", "skills"); claude != want {
		t.Fatalf("claude = %s, want %s", claude, want)
	}
	grok, err := grokPath(target)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "grok-home", "skills"); grok != want {
		t.Fatalf("grok = %s, want %s", grok, want)
	}
	copilot, err := copilotCLIPath(target)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "copilot-home", "skills"); copilot != want {
		t.Fatalf("copilot = %s, want %s", copilot, want)
	}
}

func TestResolveListExpandsBothAndAll(t *testing.T) {
	both, err := ResolveList("both")
	if err != nil {
		t.Fatal(err)
	}
	if len(both) != 2 || both[0] != "claude" || both[1] != "agents" {
		t.Fatalf("both = %v", both)
	}
	all, err := ResolveList("all")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != len(Names()) {
		t.Fatalf("all = %v, want %d hosts", all, len(Names()))
	}
	alias, err := ResolveList("github-copilot")
	if err != nil {
		t.Fatal(err)
	}
	if len(alias) != 1 || alias[0] != "copilot-cli" {
		t.Fatalf("alias = %v", alias)
	}
	if _, err := ResolveList("nope"); err == nil {
		t.Fatal("unknown agent accepted")
	}
}
