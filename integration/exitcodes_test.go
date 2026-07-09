package integration

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"
)

// runYankrun executes the built binary with args and returns combined output
// and the process exit code (0 on success).
func runYankrun(t *testing.T, bin string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = repoRoot(t)
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

func TestExitCodes(t *testing.T) {
	bin := buildBinary(t)

	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "name: [[APP_NAME]]\n")

	cases := []struct {
		name string
		args []string
		want int
	}{
		{"template missing dir", []string{"template"}, 2},
		{"template onlyTemplates without processTemplates", []string{"template", "--dir", dir, "--onlyTemplates"}, 2},
		{"clone missing repo", []string{"clone", "--outputDir", t.TempDir()}, 2},
		{"scan no target", []string{"scan"}, 2},
		{"scan both targets", []string{"scan", "--dir", dir, "--repo", "x"}, 2},
		{"scan bad input file", []string{"scan", "--dir", dir, "--input", filepath.Join(dir, "nope.yaml")}, 3},
		{"scan success", []string{"scan", "--dir", dir}, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, code := runYankrun(t, bin, tc.args...)
			if code != tc.want {
				t.Fatalf("exit code = %d, want %d\noutput:\n%s", code, tc.want, out)
			}
		})
	}
}

func TestScanJSONErrorEnvelope(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "name: [[APP_NAME]]\n")

	out, code := runYankrun(t, bin, "scan", "--dir", dir, "--input", filepath.Join(dir, "missing.yaml"), "--json")
	if code != 3 {
		t.Fatalf("exit code = %d, want 3\noutput:\n%s", code, out)
	}

	// The JSON envelope is on stdout; CombinedOutput merges stderr, so parse the
	// first JSON object by locating the opening brace.
	start := indexByte(out, '{')
	if start < 0 {
		t.Fatalf("no JSON envelope in output:\n%s", out)
	}
	var env struct {
		SchemaVersion int  `json:"schemaVersion"`
		OK            bool `json:"ok"`
		Error         struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(out[start:]), &env); err != nil {
		t.Fatalf("envelope not valid JSON: %v\noutput:\n%s", err, out)
	}
	if env.SchemaVersion != 1 || env.OK || env.Error.Code != 3 {
		t.Fatalf("unexpected envelope: %+v", env)
	}
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}
