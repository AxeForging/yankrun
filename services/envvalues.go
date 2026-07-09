package services

import (
	"os"
	"strings"
)

// EnvVarPrefix is the prefix for template values supplied through the
// environment: YANKRUN_VAR_APP_NAME=Foo sets the APP_NAME placeholder. This is
// the no-file path agents and CI use to inject values.
const EnvVarPrefix = "YANKRUN_VAR_"

// EnvValues collects template values from the process environment. The portion
// of each variable name after EnvVarPrefix becomes the placeholder key
// verbatim (placeholders are conventionally upper-snake, matching env names).
func EnvValues() map[string]string {
	return envValuesFrom(os.Environ())
}

// envValuesFrom is the testable core of EnvValues.
func envValuesFrom(environ []string) map[string]string {
	out := map[string]string{}
	for _, kv := range environ {
		if !strings.HasPrefix(kv, EnvVarPrefix) {
			continue
		}
		eq := strings.IndexByte(kv, '=')
		if eq < 0 {
			continue
		}
		key := kv[len(EnvVarPrefix):eq]
		if key == "" {
			continue
		}
		out[key] = kv[eq+1:]
	}
	return out
}
