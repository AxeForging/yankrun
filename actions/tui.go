package actions

import (
	"context"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/helpers"
	"github.com/AxeForging/yankrun/internal/tui"
	"github.com/AxeForging/yankrun/services"
	"github.com/urfave/cli/v3"
)

type TUIAction struct {
	fs       services.FileSystem
	parser   services.ReplacementParser
	replacer services.Replacer
}

func NewTUIAction(fs services.FileSystem, parser services.ReplacementParser, replacer services.Replacer) *TUIAction {
	return &TUIAction{fs: fs, parser: parser, replacer: replacer}
}

func (a *TUIAction) Execute(_ context.Context, cmd *cli.Command) error {
	dir := cmd.String("dir")
	if dir == "" {
		return helpers.UsageErr("--dir is required for tui command")
	}
	if cmd.Bool("onlyTemplates") && !cmd.Bool("processTemplates") {
		return helpers.UsageErr("--onlyTemplates requires --processTemplates to be set")
	}

	cfg, _ := services.Load()
	startDelim, endDelim, fileSizeLimit := templateSettings(cmd, cfg)

	var provided domain.InputReplacement
	if input := cmd.String("input"); input != "" {
		parsed, err := a.parser.Parse(input)
		if err != nil {
			return err
		}
		provided = parsed
	}

	return tui.Run(tui.Options{
		Dir:              dir,
		StartDelim:       startDelim,
		EndDelim:         endDelim,
		FileSizeLimit:    fileSizeLimit,
		IgnorePatterns:   append(cmd.StringSlice("ignore"), provided.IgnorePath...),
		ProcessTemplates: cmd.Bool("processTemplates"),
		OnlyTemplates:    cmd.Bool("onlyTemplates"),
		DryRun:           cmd.Bool("dryRun"),
		Verbose:          cmd.Bool("verbose"),
		Provided:         provided,
		Replacer:         a.replacer,
	})
}
