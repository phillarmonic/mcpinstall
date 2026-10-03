package copilot

import "github.com/phillarmonic/mcpinstall/platform/common"

// testOptions is the server identity the adapter tests install with. The
// adapters take identity from Options, so the tests supply a neutral one.
var testOptions = common.Options{ServerName: "test-server", Command: "test-command"}
