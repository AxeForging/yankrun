package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeManifest(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
}

func TestLoadManifest_AbsentReturnsNil(t *testing.T) {
	m, err := LoadManifest(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m != nil {
		t.Fatalf("expected nil manifest, got %+v", m)
	}
}

func TestLoadManifest_ParsesVariablesAndDefaults(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "yankrun.yaml", `version: 1
name: demo
description: A demo template
variables:
  - key: APP_NAME
    description: The application name
    required: true
  - key: ENV
    default: dev
    enum: [dev, staging, prod]
ignore_patterns:
  - "*.secret"
post_generate:
  hints:
    - run make setup
`)

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if m == nil {
		t.Fatal("expected manifest")
	}
	if m.Name != "demo" || len(m.Variables) != 2 {
		t.Fatalf("unexpected manifest: %+v", m)
	}
	if got := m.Defaults()["ENV"]; got != "dev" {
		t.Fatalf("ENV default = %q, want dev", got)
	}
	if v := m.Variable("APP_NAME"); v == nil || !v.Required {
		t.Fatalf("APP_NAME should be required: %+v", v)
	}
	if len(m.PostGenerate.Hints) != 1 {
		t.Fatalf("expected 1 hint, got %d", len(m.PostGenerate.Hints))
	}
}

func TestLoadManifest_RejectsBadPatternAndDuplicates(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "yankrun.yaml", `version: 1
variables:
  - key: X
    pattern: "([a-z"
`)
	if _, err := LoadManifest(dir); err == nil {
		t.Fatal("expected error for invalid pattern")
	}

	dir2 := t.TempDir()
	writeManifest(t, dir2, "yankrun.yaml", `version: 1
variables:
  - key: DUP
  - key: DUP
`)
	if _, err := LoadManifest(dir2); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestValidateValues(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, "yankrun.yaml", `version: 1
variables:
  - key: APP_NAME
    required: true
  - key: ENV
    enum: [dev, prod]
  - key: PORT
    pattern: "^[0-9]+$"
`)
	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	cases := []struct {
		name    string
		values  map[string]string
		wantErr string // substring, "" means no error
	}{
		{"missing required", map[string]string{"ENV": "dev"}, "APP_NAME is required"},
		{"enum rejection", map[string]string{"APP_NAME": "x", "ENV": "staging"}, "not one of"},
		{"pattern rejection", map[string]string{"APP_NAME": "x", "PORT": "abc"}, "does not match"},
		{"all valid", map[string]string{"APP_NAME": "x", "ENV": "prod", "PORT": "8080"}, ""},
		{"optional absent ok", map[string]string{"APP_NAME": "x"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateValues(m, tc.values)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestValidateValues_NilManifest(t *testing.T) {
	if err := ValidateValues(nil, map[string]string{"anything": "goes"}); err != nil {
		t.Fatalf("nil manifest should validate anything: %v", err)
	}
}
