package common

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// A commented configuration is what several hosts ship, so it has to merge
// rather than degrade to a "not valid JSON" warning.
func TestUpsertMergesJSONCConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "opencode.jsonc")
	initial := `{
    // OpenCode's schema allows comments.
    "model": "kimi-code/k3",
    /* A block comment, and a URL that must survive:
       https://opencode.ai/config.json */
    "mcp": {
        "playwright": { "type": "local" }, // trailing comment
    },
}`
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}

	changed, warning, err := UpsertMCPServerInKeyOnly(path, "test settings", "mcp", testServerName, testManagedServer(), Options{})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if warning != "" {
		t.Fatalf("a JSONC file was treated as malformed: %q", warning)
	}
	if !changed {
		t.Fatal("expected the entry to be merged")
	}

	cfg := readJSON(t, path)
	if cfg["model"] != "kimi-code/k3" {
		t.Errorf("unrelated setting was dropped: %#v", cfg)
	}
	servers := serversOf(t, cfg, "mcp")
	if _, ok := servers["playwright"]; !ok {
		t.Errorf("unrelated server was dropped: %#v", servers)
	}
	if _, ok := servers[testServerName]; !ok {
		t.Errorf("managed entry was not merged: %#v", servers)
	}
}

// The scanner must not confuse comment markers inside string literals for real
// comments, which would corrupt the value.
func TestNormalizeJSONCLeavesStringContentsAlone(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		input string
	}{
		{"URL with a double slash", `{"url":"https://example.com//path"}`},
		{"string holding a comment", `{"note":"// not a comment"}`},
		{"string holding a block comment", `{"note":"/* not a comment */"}`},
		{"escaped quote before a slash", `{"note":"a \" // b"}`},
		{"comma inside a string", `{"note":"a,}"}`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			normalized := normalizeJSONC([]byte(testCase.input))
			var want, got any
			if err := json.Unmarshal([]byte(testCase.input), &want); err != nil {
				t.Fatalf("test input is not valid JSON: %v", err)
			}
			if err := json.Unmarshal(normalized, &got); err != nil {
				t.Fatalf("normalized output is not valid JSON: %v\n%s", err, normalized)
			}
			if !jsonEqual(want, got) {
				t.Errorf("normalizeJSONC changed the value:\n in: %s\nout: %s", testCase.input, normalized)
			}
		})
	}
}

func TestNormalizeJSONCRemovesCommentsAndTrailingCommas(t *testing.T) {
	input := `{
  // a line comment
  "a": 1, /* a block comment */
  "b": [1, 2,],
  "c": {"d": 3,},
}`
	var got map[string]any
	if err := json.Unmarshal(normalizeJSONC([]byte(input)), &got); err != nil {
		t.Fatalf("normalized output is not valid JSON: %v", err)
	}
	want := map[string]any{"a": float64(1), "b": []any{float64(1), float64(2)}, "c": map[string]any{"d": float64(3)}}
	if !jsonEqual(want, got) {
		t.Errorf("normalizeJSONC = %#v, want %#v", got, want)
	}
}

func jsonEqual(a, b any) bool {
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(left) == string(right)
}
