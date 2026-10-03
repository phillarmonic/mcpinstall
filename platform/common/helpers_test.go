package common

// Test identity shared by the common-package tests. The writers take the server
// name as an argument, so these tests use a neutral name and command.
const (
	testServerName = "test-server"
	testCommand    = "test-command"
)

func testManagedServer() map[string]any {
	return ManagedServer(Options{Command: testCommand})
}
