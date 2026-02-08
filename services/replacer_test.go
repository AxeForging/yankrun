package services

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AxeForging/yankrun/domain"
)

// --- Unit tests for internal helpers ---

func TestStringToBytes(t *testing.T) {
	fr := &FileReplacer{}
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"3 mb", 3 * 1024 * 1024, false},
		{"1 kb", 1024, false},
		{"2 gb", 2 * 1024 * 1024 * 1024, false},
		{"10mb", 10 * 1024 * 1024, false},
		{"", 0, true},
		{"abc", 0, true},
		{"3 tb", 0, true},
		{"3", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := fr.stringToBytes(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("stringToBytes(%q) expected error, got %d", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("stringToBytes(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.expected {
				t.Errorf("stringToBytes(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParsePlaceholder(t *testing.T) {
	fr := &FileReplacer{}
	tests := []struct {
		input              string
		expectedKey        string
		expectedTransforms []string
		wantErr            bool
	}{
		{"APP_NAME", "APP_NAME", nil, false},
		{"APP_NAME:toLowerCase", "APP_NAME", []string{"toLowerCase"}, false},
		{"APP_NAME:gsub(-,_):toUpperCase", "APP_NAME", []string{"gsub(-,_)", "toUpperCase"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			key, transforms, err := fr.parsePlaceholder(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parsePlaceholder(%q) expected error", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parsePlaceholder(%q) unexpected error: %v", tt.input, err)
				return
			}
			if key != tt.expectedKey {
				t.Errorf("parsePlaceholder(%q) key = %q, want %q", tt.input, key, tt.expectedKey)
			}
			if len(transforms) != len(tt.expectedTransforms) {
				t.Errorf("parsePlaceholder(%q) transforms = %v, want %v", tt.input, transforms, tt.expectedTransforms)
			}
		})
	}
}

func TestApplyTransformations(t *testing.T) {
	fr := &FileReplacer{}
	tests := []struct {
		name            string
		value           string
		transformations []string
		expected        string
		wantErr         bool
	}{
		{"no transforms", "Hello", nil, "Hello", false},
		{"toUpperCase", "hello", []string{"toUpperCase"}, "HELLO", false},
		{"toLowerCase", "HELLO", []string{"toLowerCase"}, "hello", false},
		{"toDownCase", "HELLO", []string{"toDownCase"}, "hello", false},
		{"gsub", "hello world", []string{"gsub( ,-)"}, "hello-world", false},
		{"chained", "Hello World", []string{"gsub( ,-)", "toLowerCase"}, "hello-world", false},
		{"unsupported", "hello", []string{"capitalize"}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fr.applyTransformations(tt.value, tt.transformations)
			if tt.wantErr {
				if err == nil {
					t.Errorf("applyTransformations expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Errorf("applyTransformations unexpected error: %v", err)
				return
			}
			if got != tt.expected {
				t.Errorf("applyTransformations = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestApplyGsub(t *testing.T) {
	fr := &FileReplacer{}
	tests := []struct {
		name     string
		value    string
		funcStr  string
		expected string
		wantErr  bool
	}{
		{"basic", "hello world", "gsub(world,earth)", "hello earth", false},
		{"spaces to dashes", "hello world", "gsub( ,-)", "hello-world", false},
		{"empty old replaces spaces", "hello world", "gsub(,_)", "hello_world", false},
		{"invalid args", "hello", "gsub(only_one_arg)", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := fr.applyGsub(tt.value, tt.funcStr)
			if tt.wantErr {
				if err == nil {
					t.Errorf("applyGsub expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Errorf("applyGsub unexpected error: %v", err)
				return
			}
			if got != tt.expected {
				t.Errorf("applyGsub = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIsBinaryByExt(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"file.txt", false},
		{"file.go", false},
		{"file.png", true},
		{"file.PDF", true},
		{"file.exe", true},
		{"file.zip", true},
		{"file.yaml", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isBinaryByExt(tt.path); got != tt.expected {
				t.Errorf("isBinaryByExt(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestIsBinary(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{"empty", []byte{}, false},
		{"text", []byte("Hello, World!"), false},
		{"null byte", []byte("Hello\x00World"), true},
		{"newlines and tabs", []byte("Hello\n\tWorld\r\n"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBinary(tt.data); got != tt.expected {
				t.Errorf("isBinary(%q) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestShouldIgnore(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		path     string
		patterns []string
		expected bool
	}{
		{"no patterns", "/base", "/base/file.go", nil, false},
		{"match basename", "/base", "/base/file.generated.go", []string{"*.generated.go"}, true},
		{"no match", "/base", "/base/file.go", []string{"*.generated.go"}, false},
		{"empty patterns", "/base", "/base/file.go", []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldIgnore(tt.base, tt.path, tt.patterns); got != tt.expected {
				t.Errorf("shouldIgnore = %v, want %v", got, tt.expected)
			}
		})
	}
}

// mockFileInfo implements the interface needed by checkFileSize
type mockFileInfo struct {
	name    string
	size    int64
	dir     bool
	mode    fs.FileMode
	modTime time.Time
}

func (m mockFileInfo) Name() string        { return m.name }
func (m mockFileInfo) Size() int64         { return m.size }
func (m mockFileInfo) Mode() fs.FileMode   { return m.mode }
func (m mockFileInfo) IsDir() bool         { return m.dir }
func (m mockFileInfo) Sys() interface{}    { return nil }
func (m mockFileInfo) ModTime() time.Time  { return m.modTime }

func TestCheckFileSize(t *testing.T) {
	fr := &FileReplacer{}

	fi := mockFileInfo{name: "small.txt", size: 100}
	if !fr.checkFileSize(fi, 1024, false) {
		t.Error("expected small file to pass size check")
	}

	fi = mockFileInfo{name: "large.txt", size: 2048}
	if fr.checkFileSize(fi, 1024, false) {
		t.Error("expected large file to fail size check")
	}
}

// --- Integration-style tests with real filesystem ---

func TestReplaceInDir(t *testing.T) {
	dir := t.TempDir()

	content := "Hello [[NAME]], welcome to [[PROJECT]]!"
	if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	fr := &FileReplacer{FileSystem: &OsFileSystem{}}
	replacements := domain.InputReplacement{
		Variables: []domain.Replacement{
			{Key: "NAME", Value: "Alice"},
			{Key: "PROJECT", Value: "TestProject"},
		},
	}

	err := fr.ReplaceInDir(dir, replacements, "3 mb", "[[", "]]", false, nil)
	if err != nil {
		t.Fatalf("ReplaceInDir failed: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "test.txt"))
	if err != nil {
		t.Fatal(err)
	}

	expected := "Hello Alice, welcome to TestProject!"
	if string(got) != expected {
		t.Errorf("got %q, want %q", string(got), expected)
	}
}

func TestReplaceInDirWithIgnorePatterns(t *testing.T) {
	dir := t.TempDir()

	content := "Hello [[NAME]]!"
	if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skip.generated.txt"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	fr := &FileReplacer{FileSystem: &OsFileSystem{}}
	replacements := domain.InputReplacement{
		Variables: []domain.Replacement{
			{Key: "NAME", Value: "Alice"},
		},
	}

	err := fr.ReplaceInDir(dir, replacements, "3 mb", "[[", "]]", false, []string{"*.generated.txt"})
	if err != nil {
		t.Fatalf("ReplaceInDir failed: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(dir, "keep.txt"))
	if string(got) != "Hello Alice!" {
		t.Errorf("keep.txt: got %q, want %q", string(got), "Hello Alice!")
	}

	got, _ = os.ReadFile(filepath.Join(dir, "skip.generated.txt"))
	if string(got) != content {
		t.Errorf("skip.generated.txt should be unchanged, got %q", string(got))
	}
}

func TestAnalyzeDirWithIgnorePatterns(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "main.txt"), []byte("[[NAME]] [[NAME]]"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skip.gen.txt"), []byte("[[NAME]]"), 0644); err != nil {
		t.Fatal(err)
	}

	fr := &FileReplacer{FileSystem: &OsFileSystem{}}
	counts, err := fr.AnalyzeDir(dir, "3 mb", "[[", "]]", false, []string{"*.gen.txt"})
	if err != nil {
		t.Fatalf("AnalyzeDir failed: %v", err)
	}

	if counts["NAME"] != 2 {
		t.Errorf("expected NAME count 2 (skipping gen file), got %d", counts["NAME"])
	}
}

func TestAnalyzeDirCustomDelimiters(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("<!APP!> and <!VERSION!>"), 0644); err != nil {
		t.Fatal(err)
	}

	fr := &FileReplacer{FileSystem: &OsFileSystem{}}
	counts, err := fr.AnalyzeDir(dir, "3 mb", "<!", "!>", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if counts["APP"] != 1 || counts["VERSION"] != 1 {
		t.Errorf("unexpected counts: %v", counts)
	}
}

// --- Existing tests updated with ignorePatterns parameter ---

func TestProcessTemplateFiles(t *testing.T) {
	tempDir := t.TempDir()

	tplContent1 := `Hello [[NAME]]!
This is a template file with [[PROJECT_NAME]].
Version: [[VERSION:toUpperCase]]`

	tplContent2 := `Config for [[APP_NAME:toLowerCase]]:
Database: [[DB_NAME]]
Port: [[PORT]]`

	tplFile1 := filepath.Join(tempDir, "readme.tpl")
	tplFile2 := filepath.Join(tempDir, "config.tpl")
	regularFile := filepath.Join(tempDir, "regular.txt")

	if err := os.WriteFile(tplFile1, []byte(tplContent1), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(tplFile2, []byte(tplContent2), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(regularFile, []byte("This is a regular file"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	osfs := &OsFileSystem{}
	replacer := &FileReplacer{FileSystem: osfs}

	replacements := domain.InputReplacement{
		Variables: []domain.Replacement{
			{Key: "NAME", Value: "John"},
			{Key: "PROJECT_NAME", Value: "TestProject"},
			{Key: "VERSION", Value: "1.0.0"},
			{Key: "APP_NAME", Value: "MyApp"},
			{Key: "DB_NAME", Value: "testdb"},
			{Key: "PORT", Value: "8080"},
		},
	}

	err := replacer.ProcessTemplateFiles(tempDir, replacements, "3 mb", "[[", "]]", false, nil)
	if err != nil {
		t.Fatalf("ProcessTemplateFiles failed: %v", err)
	}

	if _, err := os.Stat(tplFile1); err == nil {
		t.Error("readme.tpl should have been removed")
	}
	if _, err := os.Stat(tplFile2); err == nil {
		t.Error("config.tpl should have been removed")
	}

	readmeFile := filepath.Join(tempDir, "readme")
	configFile := filepath.Join(tempDir, "config")

	if _, err := os.Stat(readmeFile); err != nil {
		t.Error("readme file should have been created")
	}
	if _, err := os.Stat(configFile); err != nil {
		t.Error("config file should have been created")
	}

	readmeContent, err := os.ReadFile(readmeFile)
	if err != nil {
		t.Fatalf("Failed to read readme file: %v", err)
	}

	expectedReadme := `Hello John!
This is a template file with TestProject.
Version: 1.0.0`

	if string(readmeContent) != expectedReadme {
		t.Errorf("readme content mismatch. Expected:\n%s\nGot:\n%s", expectedReadme, string(readmeContent))
	}

	configContent, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	expectedConfig := `Config for myapp:
Database: testdb
Port: 8080`

	if string(configContent) != expectedConfig {
		t.Errorf("config content mismatch. Expected:\n%s\nGot:\n%s", expectedConfig, string(configContent))
	}

	regularContent, err := os.ReadFile(regularFile)
	if err != nil {
		t.Fatalf("Failed to read regular file: %v", err)
	}

	if string(regularContent) != "This is a regular file" {
		t.Error("Regular file should not have been modified")
	}
}

func TestProcessTemplateFilesWithSubdirectories(t *testing.T) {
	tempDir := t.TempDir()

	subDir1 := filepath.Join(tempDir, "src")
	subDir2 := filepath.Join(tempDir, "docs")
	if err := os.MkdirAll(subDir1, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}
	if err := os.MkdirAll(subDir2, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	tplFile1 := filepath.Join(subDir1, "main.tpl")
	tplFile2 := filepath.Join(subDir2, "readme.tpl")

	if err := os.WriteFile(tplFile1, []byte(`Source file: [[FILE_NAME]]`), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(tplFile2, []byte(`Documentation: [[DOC_TITLE]]`), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	osfs := &OsFileSystem{}
	replacer := &FileReplacer{FileSystem: osfs}

	replacements := domain.InputReplacement{
		Variables: []domain.Replacement{
			{Key: "FILE_NAME", Value: "app.go"},
			{Key: "DOC_TITLE", Value: "User Guide"},
		},
	}

	err := replacer.ProcessTemplateFiles(tempDir, replacements, "3 mb", "[[", "]]", false, nil)
	if err != nil {
		t.Fatalf("ProcessTemplateFiles failed: %v", err)
	}

	if _, err := os.Stat(tplFile1); err == nil {
		t.Error("main.tpl should have been removed")
	}
	if _, err := os.Stat(tplFile2); err == nil {
		t.Error("readme.tpl should have been removed")
	}

	mainContent, err := os.ReadFile(filepath.Join(subDir1, "main"))
	if err != nil {
		t.Fatalf("Failed to read main file: %v", err)
	}
	if string(mainContent) != "Source file: app.go" {
		t.Errorf("main content mismatch. Got: '%s'", string(mainContent))
	}

	readmeContent, err := os.ReadFile(filepath.Join(subDir2, "readme"))
	if err != nil {
		t.Fatalf("Failed to read readme file: %v", err)
	}
	if string(readmeContent) != "Documentation: User Guide" {
		t.Errorf("readme content mismatch. Got: '%s'", string(readmeContent))
	}
}

func TestProcessTemplateFilesSkipsIgnoredDirectories(t *testing.T) {
	tempDir := t.TempDir()

	gitDir := filepath.Join(tempDir, ".git")
	nodeModulesDir := filepath.Join(tempDir, "node_modules")
	vendorDir := filepath.Join(tempDir, "vendor")

	for _, d := range []string{gitDir, nodeModulesDir, vendorDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	tplContent := `This should not be processed: [[VALUE]]`
	gitTplFile := filepath.Join(gitDir, "config.tpl")
	nodeTplFile := filepath.Join(nodeModulesDir, "package.tpl")
	vendorTplFile := filepath.Join(vendorDir, "lib.tpl")

	for _, f := range []string{gitTplFile, nodeTplFile, vendorTplFile} {
		if err := os.WriteFile(f, []byte(tplContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	osfs := &OsFileSystem{}
	replacer := &FileReplacer{FileSystem: osfs}

	replacements := domain.InputReplacement{
		Variables: []domain.Replacement{
			{Key: "VALUE", Value: "processed"},
		},
	}

	err := replacer.ProcessTemplateFiles(tempDir, replacements, "3 mb", "[[", "]]", false, nil)
	if err != nil {
		t.Fatalf("ProcessTemplateFiles failed: %v", err)
	}

	// .tpl files in ignored directories should still exist (not processed)
	for _, f := range []string{gitTplFile, nodeTplFile, vendorTplFile} {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("%s should not have been processed", f)
		}
	}
}

func TestProcessTemplateFilesWithTransformations(t *testing.T) {
	tempDir := t.TempDir()

	tplContent := `App: [[APP_NAME:toUpperCase]]
Path: [[PATH:gsub( ,-)]]
Mixed: [[MIXED:toLowerCase:gsub(test,prod)]]`

	tplFile := filepath.Join(tempDir, "app.tpl")
	if err := os.WriteFile(tplFile, []byte(tplContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	osfs := &OsFileSystem{}
	replacer := &FileReplacer{FileSystem: osfs}

	replacements := domain.InputReplacement{
		Variables: []domain.Replacement{
			{Key: "APP_NAME", Value: "myapp"},
			{Key: "PATH", Value: "src main"},
			{Key: "MIXED", Value: "TEST_VALUE"},
		},
	}

	err := replacer.ProcessTemplateFiles(tempDir, replacements, "3 mb", "[[", "]]", false, nil)
	if err != nil {
		t.Fatalf("ProcessTemplateFiles failed: %v", err)
	}

	if _, err := os.Stat(tplFile); err == nil {
		t.Error("app.tpl should have been removed")
	}

	appContent, err := os.ReadFile(filepath.Join(tempDir, "app"))
	if err != nil {
		t.Fatalf("Failed to read app file: %v", err)
	}

	expectedContent := `App: MYAPP
Path: src-main
Mixed: prod_value`

	if string(appContent) != expectedContent {
		t.Errorf("app content mismatch. Expected:\n%s\nGot:\n%s", expectedContent, string(appContent))
	}
}

func TestOnlyTemplatesFunctionality(t *testing.T) {
	tempDir := t.TempDir()

	tplContent := `Hello [[NAME]]!
This is a template file with [[PROJECT_NAME]].`
	regularContent := `This is a regular file with [[NAME]] placeholder.`

	tplFile := filepath.Join(tempDir, "readme.tpl")
	regularFile := filepath.Join(tempDir, "regular.txt")
	anotherRegularFile := filepath.Join(tempDir, "config.json")

	if err := os.WriteFile(tplFile, []byte(tplContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(regularFile, []byte(regularContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(anotherRegularFile, []byte(`{"name": "[[NAME]]"}`), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	osfs := &OsFileSystem{}
	replacer := &FileReplacer{FileSystem: osfs}

	replacements := domain.InputReplacement{
		Variables: []domain.Replacement{
			{Key: "NAME", Value: "John"},
			{Key: "PROJECT_NAME", Value: "TestProject"},
		},
	}

	err := replacer.ProcessTemplateFiles(tempDir, replacements, "3 mb", "[[", "]]", false, nil)
	if err != nil {
		t.Fatalf("Failed to process template files: %v", err)
	}

	processedFile := filepath.Join(tempDir, "readme")
	if _, err := os.Stat(processedFile); os.IsNotExist(err) {
		t.Error("Processed file should exist")
	}

	if _, err := os.Stat(tplFile); !os.IsNotExist(err) {
		t.Error("Original .tpl file should have been removed")
	}

	regularContentAfter, err := os.ReadFile(regularFile)
	if err != nil {
		t.Fatalf("Failed to read regular file: %v", err)
	}
	if string(regularContentAfter) != regularContent {
		t.Errorf("Regular file should not have been processed. Got: %s", string(regularContentAfter))
	}

	jsonContentAfter, err := os.ReadFile(anotherRegularFile)
	if err != nil {
		t.Fatalf("Failed to read JSON file: %v", err)
	}
	if string(jsonContentAfter) != `{"name": "[[NAME]]"}` {
		t.Errorf("JSON file should not have been processed. Got: %s", string(jsonContentAfter))
	}

	processedContent, err := os.ReadFile(processedFile)
	if err != nil {
		t.Fatalf("Failed to read processed file: %v", err)
	}
	expectedContent := `Hello John!
This is a template file with TestProject.`
	if string(processedContent) != expectedContent {
		t.Errorf("Processed content mismatch. Expected: %s, Got: %s", expectedContent, string(processedContent))
	}
}
