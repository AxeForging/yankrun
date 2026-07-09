package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runYankrunStdin runs the binary feeding stdin and extra env, returning
// combined output and the exit code.
func runYankrunStdin(t *testing.T, bin, stdin string, env []string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdin = strings.NewReader(stdin)
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

// templateEnvelope decodes a --json apply envelope from combined output.
type templateEnvelope struct {
	SchemaVersion int    `json:"schemaVersion"`
	Command       string `json:"command"`
	OK            bool   `json:"ok"`
	Data          struct {
		Applied      bool `json:"applied"`
		TotalMatches int  `json:"totalMatches"`
		Placeholders int  `json:"placeholders"`
		Summary      struct {
			Keys  []string `json:"keys"`
			Files []struct {
				Path string `json:"path"`
				Diff string `json:"diff"`
			} `json:"files"`
		} `json:"summary"`
	} `json:"data"`
}

func decodeEnvelope(t *testing.T, out string) templateEnvelope {
	t.Helper()
	start := indexByte(out, '{')
	if start < 0 {
		t.Fatalf("no JSON envelope:\n%s", out)
	}
	var env templateEnvelope
	if err := json.Unmarshal([]byte(out[start:]), &env); err != nil {
		t.Fatalf("bad envelope: %v\n%s", err, out)
	}
	return env
}

func TestTemplateJSON_ApplyAndIdempotency(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir, "app.txt", "name=[[APP_NAME]] again=[[APP_NAME]]\n")
	valsPath := writeFile(t, t.TempDir(), "v.yaml", "variables:\n  - key: APP_NAME\n    value: demo\n")

	// First apply: two matches, applied.
	out, code := runYankrun(t, bin, "template", "--dir", dir, "--input", valsPath, "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, out)
	}
	env := decodeEnvelope(t, out)
	if env.SchemaVersion != 1 || !env.OK || env.Command != "template" {
		t.Fatalf("unexpected envelope: %+v", env)
	}
	if !env.Data.Applied || env.Data.TotalMatches != 2 {
		t.Fatalf("expected applied with 2 matches, got %+v", env.Data)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "app.txt")); string(got) != "name=demo again=demo\n" {
		t.Fatalf("file not replaced: %q", string(got))
	}

	// Second apply: placeholders are gone → 0 matches, still exit 0 (idempotent).
	out2, code2 := runYankrun(t, bin, "template", "--dir", dir, "--input", valsPath, "--json")
	if code2 != 0 {
		t.Fatalf("re-run exit %d\n%s", code2, out2)
	}
	env2 := decodeEnvelope(t, out2)
	if env2.Data.TotalMatches != 0 || len(env2.Data.Summary.Keys) != 0 {
		t.Fatalf("re-run should find nothing, got %+v", env2.Data)
	}
}

func TestTemplateStdinValues(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir, "app.txt", "name=[[APP_NAME]]\n")

	out, code := runYankrunStdin(t, bin,
		`{"variables":[{"key":"APP_NAME","value":"piped"}]}`,
		nil,
		"template", "--dir", dir, "--input", "-", "--yes")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, out)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "app.txt")); string(got) != "name=piped\n" {
		t.Fatalf("stdin values not applied: %q", string(got))
	}
}

func TestTemplateEnvInjection(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir, "app.txt", "name=[[APP_NAME]]\n")

	out, code := runYankrunEnv(t, bin,
		[]string{"YANKRUN_VAR_APP_NAME=fromenv"},
		"template", "--dir", dir, "--yes")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, out)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "app.txt")); string(got) != "name=fromenv\n" {
		t.Fatalf("env value not applied: %q", string(got))
	}
}

func TestTemplateDryRunJSONDiff(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir, "app.txt", "name=[[APP_NAME]]\n")
	valsPath := writeFile(t, t.TempDir(), "v.yaml", "variables:\n  - key: APP_NAME\n    value: demo\n")

	out, code := runYankrun(t, bin, "template", "--dir", dir, "--input", valsPath, "--dryRun", "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, out)
	}
	env := decodeEnvelope(t, out)
	if env.Data.Applied {
		t.Fatal("dry run must not apply")
	}
	if len(env.Data.Summary.Files) == 0 || !strings.Contains(env.Data.Summary.Files[0].Diff, "+name=demo") {
		t.Fatalf("expected diff with new value, got %+v", env.Data.Summary.Files)
	}
	// File must be untouched on dry run.
	if got, _ := os.ReadFile(filepath.Join(dir, "app.txt")); string(got) != "name=[[APP_NAME]]\n" {
		t.Fatalf("dry run modified file: %q", string(got))
	}
}
