package actions

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/helpers"
	"github.com/AxeForging/yankrun/internal/ui"
	"github.com/AxeForging/yankrun/internal/workflow"
	"github.com/AxeForging/yankrun/services"
)

// applyOptions configures a scan → resolve → validate → apply run over a
// prepared directory. It is shared by template, clone, and generate so all
// three honor the manifest, value precedence, and validation identically.
type applyOptions struct {
	dir         string
	provided    domain.InputReplacement
	settings    workflow.TemplateSettings
	interactive bool // allow interactive prompting (human, TTY only)
	dryRun      bool
}

// runApply scans dir, resolves values by precedence (manifest defaults < file <
// env < prompts), validates them against the manifest, and applies (or previews
// on dryRun). All errors are exit-code tagged.
func runApply(engine workflow.Engine, opts applyOptions) (workflow.ApplyResult, *domain.Manifest, error) {
	summary, err := engine.ScanDir(opts.dir, opts.settings, opts.provided)
	if err != nil {
		return workflow.ApplyResult{}, nil, err
	}
	if len(summary.Keys) == 0 {
		return workflow.ApplyResult{Summary: summary}, summary.Manifest, nil
	}

	fileValues := valuesFromInput(opts.provided)
	envValues := services.EnvValues()
	answers := map[string]string{}
	if opts.interactive && helpers.IsInteractive() {
		base := workflow.ResolveValues(summary.Manifest, fileValues, envValues, nil)
		answers = promptForValues(summary.Keys, base)
	}
	resolved := workflow.ResolveValues(summary.Manifest, fileValues, envValues, answers)

	if err := services.ValidateValues(summary.Manifest, resolved); err != nil {
		return workflow.ApplyResult{}, summary.Manifest, helpers.ValidationErr("%v", err)
	}

	result, err := engine.ApplyDir(opts.dir, opts.settings, domain.InputReplacement{}, resolved, opts.dryRun, false)
	if err != nil {
		return workflow.ApplyResult{}, summary.Manifest, err
	}
	return result, summary.Manifest, nil
}

// printApply renders a human-readable apply result: the placeholder summary,
// totals, dry-run diffs, and post-generate hints.
func printApply(result workflow.ApplyResult, manifest *domain.Manifest, dryRun bool) {
	if len(result.Summary.Keys) == 0 {
		helpers.Log.Info().Msg("No placeholders found.")
		return
	}
	ui.PrintScanSummary(os.Stdout, result.Summary)
	fmt.Fprintln(os.Stdout)
	ui.PrintApplyResult(os.Stdout, result, dryRun)
	if dryRun {
		printDiffs(result)
	}
	if !dryRun && result.Applied {
		ui.PrintHints(os.Stdout, manifest)
	}
}

// parseInput parses a values file, treating an empty path as no input and "-"
// as stdin. Parse failures are validation errors (exit 3).
func parseInput(parser services.ReplacementParser, input string) (domain.InputReplacement, error) {
	if input == "" {
		return domain.InputReplacement{}, nil
	}
	parsed, err := parser.Parse(input)
	if err != nil {
		return domain.InputReplacement{}, helpers.ValidationErr("%v", err)
	}
	return parsed, nil
}

// valuesFromInput flattens an InputReplacement into a key/value map.
func valuesFromInput(in domain.InputReplacement) map[string]string {
	values := map[string]string{}
	for _, r := range in.Variables {
		values[r.Key] = r.Value
	}
	return values
}

// promptForValues asks for each key with the resolved value as the default.
// This bufio flow is the interim; the huh-based form replaces it in a later
// milestone. It only runs on a real terminal.
func promptForValues(keys []string, base map[string]string) map[string]string {
	answers := map[string]string{}
	r := bufio.NewReader(os.Stdin)
	for _, k := range keys {
		def := base[k]
		fmt.Printf("Enter value for %s [%s]: ", k, def)
		s, _ := r.ReadString('\n')
		if s = strings.TrimSpace(s); s != "" {
			answers[k] = s
		}
	}
	fmt.Println()
	return answers
}

// printDiffs renders any per-file dry-run diffs attached to the result.
func printDiffs(result workflow.ApplyResult) {
	for _, f := range result.Summary.Files {
		if f.Diff != "" {
			fmt.Fprintln(os.Stdout)
			ui.PrintDiff(os.Stdout, f.Path, f.Diff)
		}
	}
}
