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
	Counts map[string]int    `json:"counts"`
	Files  []FileSummary     `json:"files"`
	Keys   []string          `json:"keys"`
	Values map[string]string `json:"values"`
}

type FileSummary struct {
	Path   string         `json:"path"`
	Counts map[string]int `json:"counts"`
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

func (e Engine) ScanDir(dir string, settings TemplateSettings, provided domain.InputReplacement) (Summary, error) {
	files, err := e.Replacer.AnalyzeDirDetails(dir, settings.FileSizeLimit, settings.StartDelim, settings.EndDelim, settings.OnlyTemplates, settings.IgnorePatterns)
	if err != nil {
		return Summary{}, err
	}
	return Summarize(files, provided), nil
}

func (e Engine) ApplyDir(dir string, settings TemplateSettings, provided domain.InputReplacement, values map[string]string, dryRun bool, forceDryRun bool) (ApplyResult, error) {
	summary, err := e.ScanDir(dir, settings, provided)
	if err != nil {
		return ApplyResult{}, err
	}
	merged := MergeValues(summary.Values, values)
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
