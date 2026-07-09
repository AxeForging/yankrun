package services

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/AxeForging/yankrun/domain"
	"gopkg.in/yaml.v3"
)

// ManifestNames are the accepted manifest filenames at a template root, in
// priority order.
var ManifestNames = []string{"yankrun.yaml", "yankrun.yml"}

// LoadManifest reads the template manifest from dir. It returns (nil, nil) when
// no manifest file is present — a manifest is always optional. A malformed
// manifest is an error so template authors get fast feedback.
func LoadManifest(dir string) (*domain.Manifest, error) {
	for _, name := range ManifestNames {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		var m domain.Manifest
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		if err := validateManifest(&m); err != nil {
			return nil, fmt.Errorf("invalid %s: %w", name, err)
		}
		return &m, nil
	}
	return nil, nil
}

// validateManifest checks the manifest is internally consistent (compilable
// patterns, non-empty keys) before it is used to validate values.
func validateManifest(m *domain.Manifest) error {
	seen := map[string]bool{}
	for _, v := range m.Variables {
		if strings.TrimSpace(v.Key) == "" {
			return errors.New("variable with empty key")
		}
		if seen[v.Key] {
			return fmt.Errorf("duplicate variable %q", v.Key)
		}
		seen[v.Key] = true
		if v.Pattern != "" {
			if _, err := regexp.Compile(v.Pattern); err != nil {
				return fmt.Errorf("variable %q has invalid pattern: %w", v.Key, err)
			}
		}
	}
	return nil
}

// ValidateValues checks resolved values against the manifest: every required
// variable must have a value, and any value present must satisfy the variable's
// enum and pattern constraints. It returns a single error listing every
// violation so callers (and agents) see all problems at once rather than one at
// a time. A nil manifest validates anything.
func ValidateValues(m *domain.Manifest, values map[string]string) error {
	if m == nil {
		return nil
	}
	var problems []string
	for _, v := range m.Variables {
		val, present := values[v.Key]
		val = strings.TrimSpace(val)
		if v.Required && val == "" {
			problems = append(problems, fmt.Sprintf("%s is required but was not provided", v.Key))
			continue
		}
		if val == "" && !present {
			continue
		}
		if len(v.Enum) > 0 && val != "" && !contains(v.Enum, val) {
			problems = append(problems, fmt.Sprintf("%s=%q is not one of [%s]", v.Key, val, strings.Join(v.Enum, ", ")))
		}
		if v.Pattern != "" && val != "" {
			// Pattern already validated at load time, so Compile cannot fail.
			if re, err := regexp.Compile(v.Pattern); err == nil && !re.MatchString(val) {
				problems = append(problems, fmt.Sprintf("%s=%q does not match pattern %s", v.Key, val, v.Pattern))
			}
		}
	}
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("%s", strings.Join(problems, "; "))
}

func contains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
