package catalog

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestNamesAreSorted(t *testing.T) {
	want := []string{"claude", "codex", "copilot", "copilot-jetbrains", "cursor", "opencode"}
	if got := Names(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
}

func TestSelectorsAddTheMetaName(t *testing.T) {
	selectors := Selectors()
	if !slices.Contains(selectors, All) {
		t.Fatalf("Selectors() = %v, want it to offer %q", selectors, All)
	}
	for _, name := range Names() {
		if !slices.Contains(selectors, name) {
			t.Fatalf("Selectors() = %v, want it to keep %q", selectors, name)
		}
	}
	if !slices.IsSorted(selectors) {
		t.Fatalf("Selectors() = %v, want sorted order", selectors)
	}
}

func TestResolveListNormalisesInput(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"claude", Claude},
		{"Claude", Claude},
		{"  CLAUDE  ", Claude},
		{"Cursor", Cursor},
		{"copilot", Copilot},
		{"Codex", Codex},
		{"chatgpt", Codex},
		{"ChatGPT", Codex},
		{"openai", Codex},
		{"OpenAI", Codex},
		{"Copilot JetBrains", CopilotJetBrains},
		{"copilot_jetbrains", CopilotJetBrains},
		{"copilot-jetbrains", CopilotJetBrains},
	}
	for _, tc := range cases {
		got, err := ResolveList(tc.input, "")
		if err != nil {
			t.Fatalf("ResolveList(%q): %v", tc.input, err)
		}
		if !reflect.DeepEqual(got, []string{tc.want}) {
			t.Fatalf("ResolveList(%q) = %v, want [%s]", tc.input, got, tc.want)
		}
	}
}

func TestResolveListFallsBackWhenEmpty(t *testing.T) {
	got, err := ResolveList("", Codex)
	if err != nil {
		t.Fatalf("ResolveList: %v", err)
	}
	if !reflect.DeepEqual(got, []string{Codex}) {
		t.Fatalf("ResolveList(\"\") = %v, want [%s]", got, Codex)
	}
}

func TestResolveListRejectsUnknownHost(t *testing.T) {
	_, err := ResolveList("vim", "")
	if err == nil {
		t.Fatal("expected an error for an unsupported host")
	}
	if !strings.Contains(err.Error(), All) {
		t.Fatalf("error does not advertise the meta name: %v", err)
	}
}

func TestResolveListPreservesOrderAndDropsDuplicates(t *testing.T) {
	got, err := ResolveList("cursor, claude ,cursor", "")
	if err != nil {
		t.Fatalf("ResolveList: %v", err)
	}
	want := []string{Cursor, Claude}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveList = %v, want %v", got, want)
	}
}

func TestResolveListRejectsEmptyEntry(t *testing.T) {
	if _, err := ResolveList("cursor,,codex", ""); err == nil {
		t.Fatal("expected an error for an empty list entry")
	}
}

func TestResolveListHandlesSingleHost(t *testing.T) {
	got, err := ResolveList("claude", "")
	if err != nil {
		t.Fatalf("ResolveList: %v", err)
	}
	if !reflect.DeepEqual(got, []string{Claude}) {
		t.Fatalf("ResolveList = %v", got)
	}
}

func TestResolveListExpandsTheMetaName(t *testing.T) {
	every := Names()
	// Naming a host before "all" keeps it where the caller put it and drops the
	// later duplicate, so the expansion is Names() with that host moved first.
	rest := slices.DeleteFunc(slices.Clone(every), func(name string) bool { return name == Cursor })
	cursorFirst := append([]string{Cursor}, rest...)

	cases := []struct {
		input string
		want  []string
	}{
		{"all", every},
		{"ALL", every},
		{"  all  ", every},
		{"all,claude", every},
		{"claude,all", every},
		{"cursor,all", cursorFirst},
		{"all,all", every},
	}
	for _, tc := range cases {
		got, err := ResolveList(tc.input, "")
		if err != nil {
			t.Fatalf("ResolveList(%q): %v", tc.input, err)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("ResolveList(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}

	// The meta name is a valid fallback too, so a caller-supplied default of
	// "all" behaves like typing it.
	got, err := ResolveList("", All)
	if err != nil {
		t.Fatalf("ResolveList(\"\", %q): %v", All, err)
	}
	if !reflect.DeepEqual(got, every) {
		t.Fatalf("ResolveList(\"\", %q) = %v, want %v", All, got, every)
	}
}

func TestAdaptersDoNotContainTheMetaName(t *testing.T) {
	// install.Run looks adapters up by name, so a meta entry here would be
	// reported as an unsupported host rather than expanded.
	if _, ok := Adapters()[All]; ok {
		t.Fatalf("%q must not be an adapter", All)
	}
}

func TestAdapterNamesMatchCatalogKeys(t *testing.T) {
	for name, adapter := range Adapters() {
		if adapter.Name() != name {
			t.Fatalf("adapter registered as %q reports Name() = %q", name, adapter.Name())
		}
	}
}

func TestStatusesCoverEveryAdapter(t *testing.T) {
	statuses := Statuses()
	for _, name := range Names() {
		status, ok := statuses[name]
		if !ok {
			t.Fatalf("no status for %q", name)
		}
		if !status.Supported || !status.Capabilities.MCP {
			t.Fatalf("%q reports unsupported MCP capabilities: %#v", name, status)
		}
		if status.Capabilities.Format == "" || status.Capabilities.ServersKey == "" {
			t.Fatalf("%q has incomplete capabilities: %#v", name, status.Capabilities)
		}
	}
}
