package config

import (
	"os"
	"testing"
)

func TestIsTestMode(t *testing.T) {
	// Should detect test mode when running with go test
	if !isTestMode() {
		t.Error("isTestMode() should return true when running tests")
	}
}

func TestTestModeWithEnvVar(t *testing.T) {
	// Test with environment variable
	os.Setenv("TEST", "1")
	defer os.Unsetenv("TEST")

	if !isTestMode() {
		t.Error("isTestMode() should return true when TEST=1")
	}
}

func TestDefaultDatabasePath(t *testing.T) {
	config := &Config{}
	config.setDefaults()

	// Should use test.db when in test mode
	expectedPath := "./test.db"
	if config.Database.Connection != expectedPath {
		t.Errorf("Expected database connection %s, got %s", expectedPath, config.Database.Connection)
	}
}
