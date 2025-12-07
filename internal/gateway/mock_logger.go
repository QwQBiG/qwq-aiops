// +build test

package gateway

// Mock logger for testing
func mockInfo(format string, args ...interface{}) {
	// Do nothing in tests
}

// Replace logger.Info calls with mock in tests
func init() {
	// This will be used during testing
}