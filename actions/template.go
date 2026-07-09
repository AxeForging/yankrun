package actions

import (
	"context"
	"os"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/helpers"
	"github.com/AxeForging/yankrun/internal/schema"
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
	jsonOut := cmd.Bool("json")
	if jsonOut {
		helpers.SetupLogger("warn")
	}

	result, manifest, dryRun, err := t.run(cmd)

	if jsonOut {
		return schema.Emit(os.Stdout, "template", result, err)
	}
	if err != nil {
		return err
	}
	printApply(result, manifest, dryRun)
	return nil
}

func (t *TemplateAction) run(cmd *cli.Command) (workflow.ApplyResult, *domain.Manifest, bool, error) {
	dir := cmd.String("dir")
	if dir == "" {
		return workflow.ApplyResult{}, nil, false, helpers.UsageErr("--dir is required for template command")
	}
	if cmd.Bool("onlyTemplates") && !cmd.Bool("processTemplates") {
		return workflow.ApplyResult{}, nil, false, helpers.UsageErr("--onlyTemplates requires --processTemplates to be set")
	}

	cfg, _ := services.Load()
	if cfg == nil {
		cfg = &domain.Config{}
	}
	startDelim, endDelim, fileSizeLimit := templateSettings(cmd, cfg)

	provided, err := parseInput(t.parser, cmd.String("input"))
	if err != nil {
		return workflow.ApplyResult{}, nil, false, err
	}

	engine := workflow.Engine{Parser: t.parser, Replacer: t.replacer}
	dryRun := cmd.Bool("dryRun")
	result, manifest, err := runApply(engine, applyOptions{
		dir:      dir,
		provided: provided,
		settings: workflow.TemplateSettings{
			StartDelim:       startDelim,
			EndDelim:         endDelim,
			FileSizeLimit:    fileSizeLimit,
			ProcessTemplates: cmd.Bool("processTemplates"),
			OnlyTemplates:    cmd.Bool("onlyTemplates"),
			Verbose:          cmd.Bool("verbose"),
			IgnorePatterns:   append(cmd.StringSlice("ignore"), provided.IgnorePath...),
		},
		interactive: cmd.Bool("interactive") && !cmd.Bool("json") && !cmd.Bool("yes"),
		dryRun:      dryRun,
	})
	return result, manifest, dryRun, err
}
