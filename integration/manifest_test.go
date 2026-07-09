package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runYankrunEnv runs the binary with extra environment entries appended.
func runYankrunEnv(t *testing.T, bin string, env []string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return string(out), exitErr.ExitCode()
	}
	t.Fatalf("run failed to start: %v", err)
	return "", -1
}

const manifestBody = `version: 1
name: demo
variables:
  - key: APP_NAME
    description: application name
    required: true
  - key: ENV
    default: dev
    enum: [dev, staging, prod]
post_generate:
  hints:
    - run make setup
`

func manifestDir(t *testing.T) string {
	dir := t.TempDir()
	writeFile(t, dir, "yankrun.yaml", manifestBody)
	writeFile(t, dir, "app.txt", "name=[[APP_NAME]] env=[[ENV]]\n")
	return dir
}

func TestManifest_MissingRequired(t *testing.T) {
	bin := buildBinary(t)
	dir := manifestDir(t)

	out, code := runYankrun(t, bin, "template", "--dir", dir)
	if code != 3 {
		t.Fatalf("exit = %d, want 3\n%s", code, out)
	}
	if !strings.Contains(out, "APP_NAME is required") {
		t.Fatalf("missing required message:\n%s", out)
	}
}

func TestManifest_EnumRejection(t *testing.T) {
	bin := buildBinary(t)
	dir := manifestDir(t)

	out, code := runYankrunEnv(t, bin,
		[]string{"YANKRUN_VAR_APP_NAME=demo", "YANKRUN_VAR_ENV=bogus"},
		"template", "--dir", dir)
	if code != 3 {
		t.Fatalf("exit = %d, want 3\n%s", code, out)
	}
	if !strings.Contains(out, "not one of") {
		t.Fatalf("enum rejection message:\n%s", out)
	}
}

func TestManifest_DefaultApplied(t *testing.T) {
	bin := buildBinary(t)
	dir := manifestDir(t)

	// APP_NAME provided via env; ENV left unset so its manifest default applies.
	out, code := runYankrunEnv(t, bin,
		[]string{"YANKRUN_VAR_APP_NAME=demo"},
		"template", "--dir", dir)
	if code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	got, err := os.ReadFile(filepath.Join(dir, "app.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "name=demo env=dev\n" {
		t.Fatalf("unexpected content: %q", string(got))
	}
	// Post-generate hint should be surfaced.
	if !strings.Contains(out, "make setup") {
		t.Fatalf("expected post-generate hint in output:\n%s", out)
	}
}

func TestManifest_EnvOverridesFile(t *testing.T) {
	bin := buildBinary(t)
	dir := manifestDir(t)
	valsPath := writeFile(t, t.TempDir(), "values.yaml",
		"variables:\n  - key: APP_NAME\n    value: fromfile\n  - key: ENV\n    value: staging\n")

	// File sets ENV=staging; env var overrides it to prod (higher precedence).
	out, code := runYankrunEnv(t, bin,
		[]string{"YANKRUN_VAR_ENV=prod"},
		"template", "--dir", dir, "--input", valsPath)
	if code != 0 {
		t.Fatalf("exit = %d, want 0\n%s", code, out)
	}
	got, err := os.ReadFile(filepath.Join(dir, "app.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "name=fromfile env=prod\n" {
		t.Fatalf("precedence wrong: %q", string(got))
	}
}
