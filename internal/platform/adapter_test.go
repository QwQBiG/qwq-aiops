package platform

import "testing"

func TestCompatibilityDoesNotRequireOptionalDocker(t *testing.T) {
	t.Setenv("DOCKER_HOST", "")
	t.Setenv("PATH", "")
	if err := NewPlatformAdapter().ValidatePlatformCompatibility(); err != nil {
		t.Fatalf("optional container tooling prevented platform diagnostics: %v", err)
	}
}
