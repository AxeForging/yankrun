package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/services"
)

type Engine struct {
	Parser   services.ReplacementParser
	Replacer services.Replacer
	Cloner   services.Cloner
}

type TemplateSettings struct {
	StartDelim       string
	EndDelim         string
	FileSizeLimit    string
	ProcessTemplates bool
	OnlyTemplates    bool
	Verbose          bool
	IgnorePatterns   []string
}

type Summary struct {
	Counts   map[string]int    `json:"counts"`
	Files    []FileSummary     `json:"files"`
	Keys     []string          `json:"keys"`
	Values   map[string]string `json:"values"`
	Manifest *domain.Manifest  `json:"manifest,omitempty"`
}

type FileSummary struct {
	Path     string         `json:"path"`
	Counts   map[string]int `json:"counts"`
	Previews []ValuePreview `json:"previews"`
}

type ValuePreview struct {
	Key        string `json:"key"`
	Expression string `json:"expression"`
	Value      string `json:"value"`
	Count      int    `json:"count"`
	Missing    bool   `json:"missing"`
	Error      string `json:"error,omitempty"`
}

type ApplyResult struct {
	Applied      bool    `json:"applied"`
	ForcedDryRun bool    `json:"forcedDryRun"`
	TotalMatches int     `json:"totalMatches"`
	Placeholders int     `json:"placeholders"`
	Summary      Summary `json:"summary"`
}

func (e Engine) LoadInput(input string) (domain.InputReplacement, error) {
	if input == "" {
		return domain.InputReplacement{}, nil
	}
	return e.Parser.Parse(input)
}

// LoadManifest reads the optional yankrun.yaml manifest from dir. It returns
// (nil, nil) when no manifest is present.
func (e Engine) LoadManifest(dir string) (*domain.Manifest, error) {
	return services.LoadManifest(dir)
}

// withManifest loads the manifest for dir and folds its ignore patterns into
// settings. Both returned values are safe to use even when no manifest exists.
func (e Engine) withManifest(dir string, settings TemplateSettings) (TemplateSettings, *domain.Manifest, error) {
	manifest, err := e.LoadManifest(dir)
	if err != nil {
		return settings, nil, err
	}
	if manifest != nil && len(manifest.IgnorePatterns) > 0 {
		settings.IgnorePatterns = append(append([]string{}, settings.IgnorePatterns...), manifest.IgnorePatterns...)
	}
	return settings, manifest, nil
}

func (e Engine) ScanDir(dir string, settings TemplateSettings, provided domain.InputReplacement) (Summary, error) {
	settings, manifest, err := e.withManifest(dir, settings)
	if err != nil {
		return Summary{}, err
	}
	files, err := e.Replacer.AnalyzeDirDetails(dir, settings.FileSizeLimit, settings.StartDelim, settings.EndDelim, settings.OnlyTemplates, settings.IgnorePatterns)
	if err != nil {
		return Summary{}, err
	}
	summary := Summarize(files, provided)
	summary.Manifest = manifest
	applyDefaults(&summary, manifest)
	e.EvaluateSummary(&summary, files, summary.Values)
	return summary, nil
}

// applyDefaults fills manifest defaults for discovered keys that have no value
// yet. Defaults are the lowest-precedence source, so any provided value wins.
func applyDefaults(summary *Summary, manifest *domain.Manifest) {
	if manifest == nil {
		return
	}
	for k, def := range manifest.Defaults() {
		if summary.Values[k] == "" {
			summary.Values[k] = def
		}
	}
}

// ResolveValues merges value sources by precedence, lowest to highest:
// manifest defaults < input file < environment (YANKRUN_VAR_*) < interactive
// answers. Empty values never override a value already set by a lower source.
func ResolveValues(manifest *domain.Manifest, file, env, answers map[string]string) map[string]string {
	out := map[string]string{}
	if manifest != nil {
		for k, v := range manifest.Defaults() {
			out[k] = v
		}
	}
	for _, src := range []map[string]string{file, env, answers} {
		for k, v := range src {
			if v != "" {
				out[k] = v
			}
		}
	}
	return out
}

func (e Engine) ApplyDir(dir string, settings TemplateSettings, provided domain.InputReplacement, values map[string]string, dryRun bool, forceDryRun bool) (ApplyResult, error) {
	settings, manifest, err := e.withManifest(dir, settings)
	if err != nil {
		return ApplyResult{}, err
	}
	files, err := e.Replacer.AnalyzeDirDetails(dir, settings.FileSizeLimit, settings.StartDelim, settings.EndDelim, settings.OnlyTemplates, settings.IgnorePatterns)
	if err != nil {
		return ApplyResult{}, err
	}
	summary := Summarize(files, provided)
	summary.Manifest = manifest
	applyDefaults(&summary, manifest)
	merged := MergeValues(summary.Values, values)
	summary.Values = merged
	e.EvaluateSummary(&summary, files, merged)
	final := BuildFinal(summary.Keys, merged)
	result := ApplyResult{
		Applied:      false,
		ForcedDryRun: forceDryRun,
		TotalMatches: ReplacementMatchCount(summary.Keys, summary.Counts, merged),
		Placeholders: len(final.Variables),
		Summary:      summary,
	}
	if dryRun || forceDryRun || len(final.Variables) == 0 {
		return result, nil
	}
	if err := e.ApplyFinal(dir, settings, final); err != nil {
		return ApplyResult{}, err
	}
	result.Applied = true
	return result, nil
}

func (e Engine) ApplyFinal(dir string, settings TemplateSettings, final domain.InputReplacement) error {
	if !settings.OnlyTemplates {
		if err := e.Replacer.ReplaceInDir(dir, final, settings.FileSizeLimit, settings.StartDelim, settings.EndDelim, settings.Verbose, settings.IgnorePatterns); err != nil {
			return err
		}
	}
	if settings.ProcessTemplates {
		if err := e.Replacer.ProcessTemplateFiles(dir, final, settings.FileSizeLimit, settings.StartDelim, settings.EndDelim, settings.Verbose, settings.IgnorePatterns); err != nil {
			return err
		}
	}
	return nil
}

type CloneOptions struct {
	Repo      string
	Branch    string
	OutputDir string
	RemoveGit bool
	DryRun    bool
	ForceDry  bool
}

func (e Engine) CloneAndApply(opts CloneOptions, settings TemplateSettings, values map[string]string) (string, ApplyResult, error) {
	if e.Cloner == nil {
		return "", ApplyResult{}, fmt.Errorf("clone support is not configured")
	}
	if opts.Repo == "" {
		return "", ApplyResult{}, fmt.Errorf("repo is required")
	}
	workDir := opts.OutputDir
	if opts.DryRun || opts.ForceDry {
		tmp, err := os.MkdirTemp("", "yankrun-dryrun-*")
		if err != nil {
			return "", ApplyResult{}, err
		}
		defer os.RemoveAll(tmp)
		workDir = tmp
	} else if workDir == "" {
		return "", ApplyResult{}, fmt.Errorf("outputDir is required")
	}

	if opts.Branch != "" {
		if err := e.Cloner.CloneRepositoryBranch(opts.Repo, opts.Branch, workDir); err != nil {
			return "", ApplyResult{}, err
		}
	} else if err := e.Cloner.CloneRepository(opts.Repo, workDir); err != nil {
		return "", ApplyResult{}, err
	}
	if opts.RemoveGit {
		if err := os.RemoveAll(filepath.Join(workDir, ".git")); err != nil {
			return "", ApplyResult{}, err
		}
	}
	result, err := e.ApplyDir(workDir, settings, domain.InputReplacement{}, values, opts.DryRun, opts.ForceDry)
	if err != nil {
		return "", ApplyResult{}, err
	}
	return workDir, result, nil
}

func Summarize(files []services.ReplacementFile, provided domain.InputReplacement) Summary {
	counts := map[string]int{}
	fileSummaries := make([]FileSummary, 0, len(files))
	for _, file := range files {
		fileCounts := map[string]int{}
		for key, count := range file.Counts {
			counts[key] += count
			fileCounts[key] = count
		}
		fileSummaries = append(fileSummaries, FileSummary{Path: file.Path, Counts: fileCounts})
	}
	sort.Slice(fileSummaries, func(i, j int) bool {
		return fileSummaries[i].Path < fileSummaries[j].Path
	})
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	values := map[string]string{}
	for _, r := range provided.Variables {
		values[r.Key] = r.Value
	}
	return Summary{Counts: counts, Files: fileSummaries, Keys: keys, Values: values}
}

func (e Engine) EvaluateSummary(summary *Summary, files []services.ReplacementFile, values map[string]string) {
	if summary == nil || e.Replacer == nil {
		return
	}
	byPath := map[string]services.ReplacementFile{}
	for _, file := range files {
		byPath[file.Path] = file
	}
	for i := range summary.Files {
		file, ok := byPath[summary.Files[i].Path]
		if !ok {
			continue
		}
		previews := make([]ValuePreview, 0, len(file.Placeholders))
		for _, occurrence := range file.Placeholders {
			value, found, err := e.Replacer.EvaluatePlaceholder(occurrence.Expression, values)
			preview := ValuePreview{
				Key:        occurrence.Key,
				Expression: occurrence.Expression,
				Value:      value,
				Count:      occurrence.Count,
				Missing:    !found,
			}
			if err != nil {
				preview.Error = err.Error()
			}
			previews = append(previews, preview)
		}
		sort.Slice(previews, func(a, b int) bool {
			if previews[a].Key == previews[b].Key {
				return previews[a].Expression < previews[b].Expression
			}
			return previews[a].Key < previews[b].Key
		})
		summary.Files[i].Previews = previews
	}
}

func MergeValues(base map[string]string, override map[string]string) map[string]string {
	merged := map[string]string{}
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range override {
		merged[k] = v
	}
	return merged
}

func BuildFinal(keys []string, values map[string]string) domain.InputReplacement {
	final := domain.InputReplacement{}
	for _, k := range keys {
		if v, ok := values[k]; ok && v != "" {
			final.Variables = append(final.Variables, domain.Replacement{Key: k, Value: v})
		}
	}
	return final
}

func ReplacementMatchCount(keys []string, counts map[string]int, values map[string]string) int {
	total := 0
	for _, k := range keys {
		if v, ok := values[k]; ok && v != "" {
			total += counts[k]
		}
	}
	return total
}
