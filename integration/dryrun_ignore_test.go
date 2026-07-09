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
	if err := os.WriteFile(filepath.Join(testDir, "test.txt"), []byte(content), 0o644); err != nil {
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
	testDir := filepath.Join(t.TempDir(), "out")

	// Create a local git repo to clone from
	repoDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoDir, "readme.txt"), []byte("Hello [[NAME]]!"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("git", "init", "-b", "main")
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

	// Clone dry-run should not leave output files behind.
	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Fatalf("dry-run clone should not create output dir, stat err=%v", err)
	}
}

func TestCloneBranchFlag(t *testing.T) {
	bin := buildBinary(t)
	outDir := filepath.Join(t.TempDir(), "out")
	repoDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(repoDir, "readme.txt"), []byte("main branch"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, string(out))
	}
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, string(out))
	}
	cmd = exec.Command("git", "commit", "-m", "main")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, string(out))
	}
	cmd = exec.Command("git", "switch", "-c", "feature-template")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git switch failed: %v\n%s", err, string(out))
	}
	if err := os.WriteFile(filepath.Join(repoDir, "readme.txt"), []byte("feature [[NAME]]"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, string(out))
	}
	cmd = exec.Command("git", "commit", "-m", "feature")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@test.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@test.com")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, string(out))
	}

	valsPath := writeFile(t, t.TempDir(), "values.yaml", `variables: [{key: NAME, value: World}]`)
	cmd = exec.Command(bin,
		"clone",
		"--repo", repoDir,
		"--outputDir", outDir,
		"--branch", "feature-template",
		"--input", valsPath,
	)
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("clone branch failed: %v\n%s", err, string(out))
	}

	got, err := os.ReadFile(filepath.Join(outDir, "readme.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "feature World" {
		t.Fatalf("readme.txt = %q, want feature branch content", string(got))
	}
}

func TestGenerateDryRunDoesNotCreateOutput(t *testing.T) {
	bin := buildBinary(t)
	home := t.TempDir()
	configDir := filepath.Join(home, ".yankrun")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}

	repoDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoDir, "readme.txt"), []byte("Hello [[NAME]]"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", "-b", "main")
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

	config := "templates:\n  - name: local\n    url: " + repoDir + "\n    default_branch: main\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	valsPath := writeFile(t, t.TempDir(), "values.yaml", `variables: [{key: NAME, value: World}]`)
	outDir := filepath.Join(t.TempDir(), "generated")

	cmd = exec.Command(bin,
		"generate",
		"--template", "local",
		"--outputDir", outDir,
		"--input", valsPath,
		"--dryRun",
	)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "HOME="+home)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generate dry-run failed: %v\n%s", err, string(out))
	}
	if !strings.Contains(string(out), "Dry run") {
		t.Fatalf("expected dry-run output, got:\n%s", string(out))
	}
	if _, err := os.Stat(outDir); !os.IsNotExist(err) {
		t.Fatalf("generate dry-run should not create output dir, stat err=%v", err)
	}
}

func TestTemplateUsesConfigDefaults(t *testing.T) {
	bin := buildBinary(t)
	home := t.TempDir()
	configDir := filepath.Join(home, ".yankrun")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	config := "start_delim: '<%'\nend_delim: '%>'\nfile_size_limit: '1 mb'\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}

	testDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(testDir, "app.txt"), []byte("App: <%NAME%>"), 0o644); err != nil {
		t.Fatal(err)
	}
	valsPath := writeFile(t, t.TempDir(), "values.yaml", `variables: [{key: NAME, value: MyApp}]`)

	cmd := exec.Command(bin, "template", "--dir", testDir, "--input", valsPath)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "HOME="+home)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("template with config defaults failed: %v\n%s", err, string(out))
	}

	got, err := os.ReadFile(filepath.Join(testDir, "app.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "App: MyApp" {
		t.Fatalf("app.txt = %q, want config delimiters to apply", string(got))
	}
}

// The TUI now requires a real terminal (it is a full-screen Bubble Tea app).
// Driven non-interactively — as agents and CI do — it must refuse with a usage
// error rather than hang or corrupt the terminal. The interactive behavior
// (scan → fill → preview → apply) is covered by the teatest suite in
// internal/tui.
func TestTUIRequiresTTY(t *testing.T) {
	bin := buildBinary(t)
	testDir := t.TempDir()
	path := filepath.Join(testDir, "app.txt")
	if err := os.WriteFile(path, []byte("App: [[NAME]]"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "tui", "--dir", testDir)
	cmd.Dir = repoRoot(t)
	cmd.Stdin = strings.NewReader("")
	out, err := cmd.CombinedOutput()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	}
	if code != 2 {
		t.Fatalf("tui without a TTY: exit %d, want 2\n%s", code, string(out))
	}
	if got, _ := os.ReadFile(path); string(got) != "App: [[NAME]]" {
		t.Fatalf("tui must not modify files: %q", string(got))
	}
}

func TestIgnorePatternsFlag(t *testing.T) {
	bin := buildBinary(t)
	testDir := t.TempDir()

	// Create test files
	if err := os.WriteFile(filepath.Join(testDir, "main.txt"), []byte("Hello [[NAME]]!"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "generated.lock"), []byte("Lock: [[NAME]]"), 0o644); err != nil {
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
	if err := os.WriteFile(filepath.Join(testDir, "app.txt"), []byte("App: [[NAME]]"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "skip.min.js"), []byte("var x='[[NAME]]'"), 0o644); err != nil {
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
