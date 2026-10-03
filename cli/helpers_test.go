package cli

import (
	"os"

	"github.com/phillarmonic/mcpinstall"
)

// testSpec is the server identity used by the CLI tests. Guidance is read from
// testdata/guidance so the tests do not depend on any project's files.
var testSpec = mcpinstall.Spec{
	Name:     "test-server",
	Command:  "test-command",
	Program:  "test",
	Marker:   "test",
	Guidance: os.DirFS("testdata"),
}
