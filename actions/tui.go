package actions

import (
	"fmt"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/internal/tui"
	"github.com/AxeForging/yankrun/services"
	"github.com/urfave/cli"
)

type TUIAction struct {
	fs       services.FileSystem
	parser   services.ReplacementParser
	replacer services.Replacer
}

func NewTUIAction(fs services.FileSystem, parser services.ReplacementParser, replacer services.Replacer) *TUIAction {
	return &TUIAction{fs: fs, parser: parser, replacer: replacer}
}

func (a *TUIAction) Execute(c *cli.Context) error {
	dir := c.String("dir")
	if dir == "" {
		return fmt.Errorf("--dir is required for tui command")
	}
	if c.Bool("onlyTemplates") && !c.Bool("processTemplates") {
		return fmt.Errorf("--onlyTemplates requires --processTemplates to be set")
	}

	cfg, _ := services.Load()
	startDelim, endDelim, fileSizeLimit := templateSettings(c, cfg)

	var provided domain.InputReplacement
	if input := c.String("input"); input != "" {
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
		IgnorePatterns:   append(c.StringSlice("ignore"), provided.IgnorePath...),
		ProcessTemplates: c.Bool("processTemplates"),
		OnlyTemplates:    c.Bool("onlyTemplates"),
		DryRun:           c.Bool("dryRun"),
		Verbose:          c.Bool("verbose"),
		Provided:         provided,
		Replacer:         a.replacer,
	})
}
