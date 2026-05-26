package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCreatesConfigDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load returned nil config")
	}

	path := filepath.Join(home, ".yankrun", "config.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
}

func TestResetIgnoresMissingConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := Reset(); err != nil {
		t.Fatalf("Reset failed for missing config: %v", err)
	}
}
