package app

import (
	"os"
	"testing"
)

func TestLoadConfig_Missing(t *testing.T) {
	os.Unsetenv("JENKINS_USERNAME")
	os.Unsetenv("JENKINS_API_TOKEN")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error when env vars are missing, got nil")
	}
}

func TestLoadConfig_Success(t *testing.T) {
	os.Setenv("JENKINS_USERNAME", "testuser")
	os.Setenv("JENKINS_API_TOKEN", "testtoken")
	defer os.Unsetenv("JENKINS_USERNAME")
	defer os.Unsetenv("JENKINS_API_TOKEN")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if cfg.Username != "testuser" || cfg.APIToken != "testtoken" {
		t.Error("config values not set correctly")
	}
}
