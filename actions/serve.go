package actions

import (
	"fmt"
	"net"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/helpers"
	"github.com/AxeForging/yankrun/internal/web"
	"github.com/AxeForging/yankrun/services"
	"github.com/urfave/cli"
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

func (a *ServeAction) Execute(c *cli.Context) error {
	dir := c.String("dir")
	if dir == "" {
		return fmt.Errorf("--dir is required for serve command")
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

	addr := c.String("addr")
	if addr == "" {
		addr = "127.0.0.1:17817"
	}
	if host, _, err := net.SplitHostPort(addr); err == nil && host == "" {
		addr = "127.0.0.1" + addr
	}

	server, err := web.New(web.Options{
		Addr:             addr,
		Dir:              dir,
		Input:            c.String("input"),
		StartDelim:       startDelim,
		EndDelim:         endDelim,
		FileSizeLimit:    fileSizeLimit,
		IgnorePatterns:   append(c.StringSlice("ignore"), provided.IgnorePath...),
		ProcessTemplates: c.Bool("processTemplates"),
		OnlyTemplates:    c.Bool("onlyTemplates"),
		ForceDryRun:      c.Bool("dryRun"),
		Verbose:          c.Bool("verbose"),
		Parser:           a.parser,
		Replacer:         a.replacer,
		Cloner:           a.cloner,
		Config:           cfg,
	})
	if err != nil {
		return fmt.Errorf("init web server: %w", err)
	}

	helpers.Log.Info().Msgf("YankRun web UI listening on http://%s", server.Addr())
	fmt.Printf("\n  Open http://%s in your browser.\n\n", server.Addr())
	return server.ListenAndServe()
}
