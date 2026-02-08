package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseYAML(t *testing.T) {
	dir := t.TempDir()
	content := `variables:
  - key: NAME
    value: Alice
  - key: PROJECT
    value: TestProject
ignore_patterns:
  - "*.generated.go"
  - "vendor/*"`

	path := filepath.Join(dir, "values.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	parser := &YAMLJSONParser{FileSystem: &OsFileSystem{}}
	result, err := parser.Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(result.Variables) != 2 {
		t.Fatalf("expected 2 variables, got %d", len(result.Variables))
	}
	if result.Variables[0].Key != "NAME" || result.Variables[0].Value != "Alice" {
		t.Errorf("unexpected first variable: %+v", result.Variables[0])
	}
	if len(result.IgnorePath) != 2 {
		t.Fatalf("expected 2 ignore patterns, got %d", len(result.IgnorePath))
	}
	if result.IgnorePath[0] != "*.generated.go" {
		t.Errorf("unexpected ignore pattern: %s", result.IgnorePath[0])
	}
}

func TestParseJSON(t *testing.T) {
	dir := t.TempDir()
	content := `{"variables": [{"key": "APP", "value": "MyApp"}], "ignore_patterns": ["*.lock"]}`

	path := filepath.Join(dir, "values.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	parser := &YAMLJSONParser{FileSystem: &OsFileSystem{}}
	result, err := parser.Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(result.Variables) != 1 || result.Variables[0].Key != "APP" {
		t.Errorf("unexpected variables: %+v", result.Variables)
	}
	if len(result.IgnorePath) != 1 || result.IgnorePath[0] != "*.lock" {
		t.Errorf("unexpected ignore patterns: %v", result.IgnorePath)
	}
}

func TestParseYML(t *testing.T) {
	dir := t.TempDir()
	content := `variables:
  - key: X
    value: Y`

	path := filepath.Join(dir, "values.yml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	parser := &YAMLJSONParser{FileSystem: &OsFileSystem{}}
	result, err := parser.Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(result.Variables) != 1 {
		t.Errorf("expected 1 variable, got %d", len(result.Variables))
	}
}

func TestParseUnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "values.toml")
	if err := os.WriteFile(path, []byte("key = 'value'"), 0644); err != nil {
		t.Fatal(err)
	}

	parser := &YAMLJSONParser{FileSystem: &OsFileSystem{}}
	_, err := parser.Parse(path)
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestParseInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	parser := &YAMLJSONParser{FileSystem: &OsFileSystem{}}
	_, err := parser.Parse(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestParseInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{invalid json"), 0644); err != nil {
		t.Fatal(err)
	}

	parser := &YAMLJSONParser{FileSystem: &OsFileSystem{}}
	_, err := parser.Parse(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseNonExistentFile(t *testing.T) {
	parser := &YAMLJSONParser{FileSystem: &OsFileSystem{}}
	_, err := parser.Parse("/nonexistent/path/values.yaml")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestParseEmptyVariables(t *testing.T) {
	dir := t.TempDir()
	content := `variables: []`

	path := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	parser := &YAMLJSONParser{FileSystem: &OsFileSystem{}}
	result, err := parser.Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(result.Variables) != 0 {
		t.Errorf("expected 0 variables, got %d", len(result.Variables))
	}
}
