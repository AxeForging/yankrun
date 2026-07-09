package actions

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/helpers"
	"github.com/AxeForging/yankrun/internal/ui"
	"github.com/AxeForging/yankrun/internal/workflow"
	"github.com/AxeForging/yankrun/services"

	"github.com/urfave/cli/v3"
)

type TemplateAction struct {
	fs       services.FileSystem
	parser   services.ReplacementParser
	replacer services.Replacer
}

func NewTemplateAction(fs services.FileSystem, parser services.ReplacementParser, replacer services.Replacer) *TemplateAction {
	return &TemplateAction{fs: fs, parser: parser, replacer: replacer}
}

func (t *TemplateAction) Execute(_ context.Context, cmd *cli.Command) error {
	dir := cmd.String("dir")
	if dir == "" {
		return helpers.UsageErr("--dir is required for template command")
	}
	if cmd.Bool("onlyTemplates") && !cmd.Bool("processTemplates") {
		return helpers.UsageErr("--onlyTemplates requires --processTemplates to be set")
	}

	cfg, _ := services.Load()
	if cfg == nil {
		cfg = &domain.Config{}
	}
	startDelim, endDelim, fileSizeLimit := templateSettings(cmd, cfg)

	// Parse the optional values file.
	var provided domain.InputReplacement
	if inputFile := cmd.String("input"); inputFile != "" {
		parsed, err := t.parser.Parse(inputFile)
		if err != nil {
			return helpers.ValidationErr("%v", err)
		}
		provided = parsed
	}

	engine := workflow.Engine{Parser: t.parser, Replacer: t.replacer}
	settings := workflow.TemplateSettings{
		StartDelim:       startDelim,
		EndDelim:         endDelim,
		FileSizeLimit:    fileSizeLimit,
		ProcessTemplates: cmd.Bool("processTemplates"),
		OnlyTemplates:    cmd.Bool("onlyTemplates"),
		Verbose:          cmd.Bool("verbose"),
		IgnorePatterns:   append(cmd.StringSlice("ignore"), provided.IgnorePath...),
	}

	// Scan first so we know the discovered keys and any manifest metadata.
	summary, err := engine.ScanDir(dir, settings, provided)
	if err != nil {
		return err
	}
	if len(summary.Keys) == 0 {
		helpers.Log.Info().Msg("No placeholders found.")
		return nil
	}
	ui.PrintScanSummary(os.Stdout, summary)

	// Resolve values by precedence: manifest defaults < file < env < prompts.
	fileValues := valuesFromInput(provided)
	envValues := services.EnvValues()
	answers := map[string]string{}
	if cmd.Bool("interactive") && helpers.IsInteractive() {
		base := workflow.ResolveValues(summary.Manifest, fileValues, envValues, nil)
		answers = promptForValues(summary.Keys, base)
	}
	resolved := workflow.ResolveValues(summary.Manifest, fileValues, envValues, answers)

	// Validate against the manifest before touching any files.
	if err := services.ValidateValues(summary.Manifest, resolved); err != nil {
		return helpers.ValidationErr("%v", err)
	}

	dryRun := cmd.Bool("dryRun")
	result, err := engine.ApplyDir(dir, settings, domain.InputReplacement{}, resolved, dryRun, false)
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout)
	ui.PrintApplyResult(os.Stdout, result, dryRun)
	if !dryRun && result.Applied {
		ui.PrintHints(os.Stdout, summary.Manifest)
	}
	return nil
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
