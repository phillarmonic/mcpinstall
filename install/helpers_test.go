package install

import (
	"os"

	"github.com/phillarmonic/mcpinstall"
)

const testMarker = "test"

// testBegin and testEnd are the managed markers derived from the test spec.
var testBegin, testEnd = managedMarkers(testMarker)

// testSpec is the server identity used by the install tests. Guidance is read
// from testdata/guidance so the tests do not depend on any project's files.
var testSpec = mcpinstall.Spec{
	Name:     "test-server",
	Command:  "test-command",
	Program:  "test",
	Marker:   testMarker,
	Guidance: os.DirFS("testdata"),
}
