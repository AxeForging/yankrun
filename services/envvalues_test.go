package services

import "testing"

func TestEnvValuesFrom(t *testing.T) {
	environ := []string{
		"PATH=/usr/bin",
		"YANKRUN_VAR_APP_NAME=Widget",
		"YANKRUN_VAR_USER_EMAIL=a@b.com",
		"YANKRUN_VAR_=ignored-empty-key",
		"YANKRUN_INPUT=values.yaml", // not a VAR_ prefix
		"YANKRUN_VAR_WITH_EQ=a=b=c", // value may contain '='
	}
	got := envValuesFrom(environ)

	want := map[string]string{
		"APP_NAME":   "Widget",
		"USER_EMAIL": "a@b.com",
		"WITH_EQ":    "a=b=c",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d values, want %d: %+v", len(got), len(want), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("%s = %q, want %q", k, got[k], v)
		}
	}
}
