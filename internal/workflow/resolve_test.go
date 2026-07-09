package workflow

import (
	"testing"

	"github.com/AxeForging/yankrun/domain"
)

func TestResolveValues_Precedence(t *testing.T) {
	manifest := &domain.Manifest{
		Variables: []domain.ManifestVar{
			{Key: "APP_NAME", Default: "default-app"},
			{Key: "ENV", Default: "dev"},
			{Key: "REGION", Default: "us"},
			{Key: "OWNER", Default: "nobody"},
		},
	}
	file := map[string]string{"APP_NAME": "from-file", "ENV": "from-file"}
	env := map[string]string{"ENV": "from-env", "REGION": "from-env"}
	answers := map[string]string{"REGION": "from-answer", "OWNER": ""} // empty must not override

	got := ResolveValues(manifest, file, env, answers)

	want := map[string]string{
		"APP_NAME": "from-file",   // default < file
		"ENV":      "from-env",    // file < env
		"REGION":   "from-answer", // env < answer
		"OWNER":    "nobody",      // empty answer keeps default
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("%s = %q, want %q (full: %+v)", k, got[k], v, got)
		}
	}
}

func TestResolveValues_NilManifest(t *testing.T) {
	got := ResolveValues(nil, map[string]string{"A": "1"}, nil, nil)
	if got["A"] != "1" || len(got) != 1 {
		t.Fatalf("unexpected: %+v", got)
	}
}
