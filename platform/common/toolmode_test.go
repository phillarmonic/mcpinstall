package common

import "testing"

func TestValidateToolModeAcceptsKnownModes(t *testing.T) {
	for _, mode := range []string{"", ToolModeCodeMode, ToolModeNative, "codemode", "native-tools"} {
		if err := ValidateToolMode(mode); err != nil {
			t.Fatalf("ValidateToolMode(%q) = %v, want nil", mode, err)
		}
	}
}

func TestValidateToolModeRejectsUnknownModes(t *testing.T) {
	err := ValidateToolMode("magic")
	if err == nil {
		t.Fatal("expected an error for an unknown tool mode")
	}
	if got := err.Error(); got == "" {
		t.Fatal("the error should name the supported modes")
	}
}

func TestNativeToolsFollowsTheMode(t *testing.T) {
	cases := []struct {
		mode string
		want bool
	}{
		{"", false},
		{ToolModeCodeMode, false},
		{"codemode", false},
		{ToolModeNative, true},
		{" NATIVE ", true},
		{"native-tools", true},
	}
	for _, tc := range cases {
		if got := NativeTools(Options{ToolMode: tc.mode}); got != tc.want {
			t.Fatalf("NativeTools(%q) = %v, want %v", tc.mode, got, tc.want)
		}
	}
}
