package actions

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/helpers"
	"github.com/AxeForging/yankrun/internal/ui"
	"github.com/AxeForging/yankrun/internal/web"
	"github.com/AxeForging/yankrun/services"
	"github.com/urfave/cli/v3"
)

type ServeAction struct {
	fs       services.FileSystem
	parser   services.ReplacementParser
	replacer services.Replacer
	cloner   services.Cloner
}

func NewServeAction(fs services.FileSystem, parser services.ReplacementParser, replacer services.Replacer, cloner services.Cloner) *ServeAction {
	return &ServeAction{fs: fs, parser: parser, replacer: replacer, cloner: cloner}
}

func (a *ServeAction) Execute(_ context.Context, cmd *cli.Command) error {
	dir := cmd.String("dir")
	if dir == "" {
		return helpers.UsageErr("--dir is required for serve command")
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

	addr := cmd.String("addr")
	if addr == "" {
		addr = "127.0.0.1:17817"
	}
	if host, _, err := net.SplitHostPort(addr); err == nil && host == "" {
		addr = "127.0.0.1" + addr
	}

	server, err := web.New(web.Options{
		Addr:             addr,
		Dir:              dir,
		Input:            cmd.String("input"),
		StartDelim:       startDelim,
		EndDelim:         endDelim,
		FileSizeLimit:    fileSizeLimit,
		IgnorePatterns:   append(cmd.StringSlice("ignore"), provided.IgnorePath...),
		ProcessTemplates: cmd.Bool("processTemplates"),
		OnlyTemplates:    cmd.Bool("onlyTemplates"),
		ForceDryRun:      cmd.Bool("dryRun"),
		Verbose:          cmd.Bool("verbose"),
		Parser:           a.parser,
		Replacer:         a.replacer,
		Cloner:           a.cloner,
		Config:           cfg,
	})
	if err != nil {
		return fmt.Errorf("init web server: %w", err)
	}

	url := "http://" + server.Addr()
	helpers.Log.Info().Msgf("YankRun web UI listening on %s", url)
	ui.PrintServeBanner(os.Stdout, url)
	return server.ListenAndServe()
}
