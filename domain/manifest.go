package domain

// Manifest is an optional yankrun.yaml at a template's root that declares the
// variables a template expects. It powers richer prompts (TUI, web, huh forms),
// pre-apply validation, and agent self-discovery via `scan --json`. A template
// without a manifest keeps working exactly as before — the manifest only adds
// metadata and guard rails.
type Manifest struct {
	Version        int           `yaml:"version" json:"version"`
	Name           string        `yaml:"name,omitempty" json:"name,omitempty"`
	Description    string        `yaml:"description,omitempty" json:"description,omitempty"`
	Variables      []ManifestVar `yaml:"variables" json:"variables"`
	IgnorePatterns []string      `yaml:"ignore_patterns,omitempty" json:"ignorePatterns,omitempty"`
	PostGenerate   PostGenerate  `yaml:"post_generate,omitempty" json:"postGenerate,omitempty"`
}

// ManifestVar describes a single template variable.
type ManifestVar struct {
	Key         string   `yaml:"key" json:"key"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Default     string   `yaml:"default,omitempty" json:"default,omitempty"`
	Required    bool     `yaml:"required,omitempty" json:"required,omitempty"`
	Enum        []string `yaml:"enum,omitempty" json:"enum,omitempty"`
	Pattern     string   `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	Transforms  []string `yaml:"transforms,omitempty" json:"transforms,omitempty"`
}

// PostGenerate carries guidance shown after a successful generate/apply, e.g.
// "run `make setup`". Hints are informational only.
type PostGenerate struct {
	Hints []string `yaml:"hints,omitempty" json:"hints,omitempty"`
}

// Variable returns the manifest entry for key, or nil when the manifest does
// not describe it.
func (m *Manifest) Variable(key string) *ManifestVar {
	if m == nil {
		return nil
	}
	for i := range m.Variables {
		if m.Variables[i].Key == key {
			return &m.Variables[i]
		}
	}
	return nil
}

// Defaults returns the declared default values keyed by variable name. Only
// variables with a non-empty default are included.
func (m *Manifest) Defaults() map[string]string {
	out := map[string]string{}
	if m == nil {
		return out
	}
	for _, v := range m.Variables {
		if v.Default != "" {
			out[v.Key] = v.Default
		}
	}
	return out
}
