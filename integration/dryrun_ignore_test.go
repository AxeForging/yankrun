package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDryRunTemplate(t *testing.T) {
	bin := buildBinary(t)
	testDir := t.TempDir()

	// Create a test file with placeholders
	content := "Hello [[NAME]], version [[VERSION]]!"
	if err := os.WriteFile(filepath.Join(testDir, "test.txt"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	vals := `variables:
  - key: NAME
    value: World
  - key: VERSION
    value: 1.0.0`
	valsPath := writeFile(t, t.TempDir(), "values.yaml", vals)

	cmd := exec.Command(bin,
		"template",
		"--dir", testDir,
		"--input", valsPath,
		"--startDelim", "[[",
		"--endDelim", "]]",
		"--dryRun",
	)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dry-run template failed: %v\n%s", err, string(out))
	}

	// Output should mention "Dry run"
	if !strings.Contains(string(out), "Dry run") {
		t.Errorf("expected 'Dry run' in output, got:\n%s", string(out))
	}

	// File should NOT be modified (dry run)
	got, err := os.ReadFile(filepath.Join(testDir, "test.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Errorf("dry-run should not modify files. Got: %q, Want: %q", string(got), content)
	}
}

func TestDryRunClone(t *testing.T) {
	bin := buildBinary(t)
	testDir := t.TempDir()

	// Create a local git repo to clone from
	repoDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoDir, "readme.txt"), []byte("Hello [[NAME]]!"), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, string(out))
	}
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, string(out))
	}
	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, string(out))
	}

	vals := `variables:
  - key: NAME
    value: World`
	valsPath := writeFile(t, t.TempDir(), "values.yaml", vals)

	cmd = exec.Command(bin,
		"clone",
		"--repo", repoDir,
		"--outputDir", testDir,
		"--input", valsPath,
		"--startDelim", "[[",
		"--endDelim", "]]",
		"--dryRun",
	)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dry-run clone failed: %v\n%s", err, string(out))
	}

	if !strings.Contains(string(out), "Dry run") {
		t.Errorf("expected 'Dry run' in output, got:\n%s", string(out))
	}

	// File should still have placeholder (dry run skips replacement)
	got, err := os.ReadFile(filepath.Join(testDir, "readme.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "[[NAME]]") {
		t.Errorf("dry-run should not replace placeholders. Got: %q", string(got))
	}
}

func TestIgnorePatternsFlag(t *testing.T) {
	bin := buildBinary(t)
	testDir := t.TempDir()

	// Create test files
	if err := os.WriteFile(filepath.Join(testDir, "main.txt"), []byte("Hello [[NAME]]!"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "generated.lock"), []byte("Lock: [[NAME]]"), 0644); err != nil {
		t.Fatal(err)
	}

	vals := `variables:
  - key: NAME
    value: World`
	valsPath := writeFile(t, t.TempDir(), "values.yaml", vals)

	cmd := exec.Command(bin,
		"template",
		"--dir", testDir,
		"--input", valsPath,
		"--startDelim", "[[",
		"--endDelim", "]]",
		"--ignore", "*.lock",
	)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("template with ignore failed: %v\n%s", err, string(out))
	}

	// main.txt should be replaced
	got, _ := os.ReadFile(filepath.Join(testDir, "main.txt"))
	if string(got) != "Hello World!" {
		t.Errorf("main.txt should be replaced. Got: %q", string(got))
	}

	// generated.lock should be unchanged
	got, _ = os.ReadFile(filepath.Join(testDir, "generated.lock"))
	if string(got) != "Lock: [[NAME]]" {
		t.Errorf("generated.lock should be unchanged. Got: %q", string(got))
	}
}

func TestIgnorePatternsFromValuesFile(t *testing.T) {
	bin := buildBinary(t)
	testDir := t.TempDir()

	// Create test files
	if err := os.WriteFile(filepath.Join(testDir, "app.txt"), []byte("App: [[NAME]]"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "skip.min.js"), []byte("var x='[[NAME]]'"), 0644); err != nil {
		t.Fatal(err)
	}

	vals := `variables:
  - key: NAME
    value: MyApp
ignore_patterns:
  - "*.min.js"`
	valsPath := writeFile(t, t.TempDir(), "values.yaml", vals)

	cmd := exec.Command(bin,
		"template",
		"--dir", testDir,
		"--input", valsPath,
		"--startDelim", "[[",
		"--endDelim", "]]",
	)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("template failed: %v\n%s", err, string(out))
	}

	// app.txt should be replaced
	got, _ := os.ReadFile(filepath.Join(testDir, "app.txt"))
	if string(got) != "App: MyApp" {
		t.Errorf("app.txt should be replaced. Got: %q", string(got))
	}

	// skip.min.js should be unchanged (ignore_patterns in values file)
	got, _ = os.ReadFile(filepath.Join(testDir, "skip.min.js"))
	if string(got) != "var x='[[NAME]]'" {
		t.Errorf("skip.min.js should be unchanged. Got: %q", string(got))
	}
}
