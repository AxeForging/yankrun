package actions

import (
	"context"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/internal/mcp"
	"github.com/AxeForging/yankrun/services"
	"github.com/urfave/cli/v3"
)

// MCPAction serves yankrun's templating engine over the Model Context Protocol
// on stdio, so agents can scan/preview/apply templates as native tools.
type MCPAction struct {
	parser   services.ReplacementParser
	replacer services.Replacer
	cloner   services.Cloner
	version  string
}

func NewMCPAction(parser services.ReplacementParser, replacer services.Replacer, cloner services.Cloner, version string) *MCPAction {
	return &MCPAction{parser: parser, replacer: replacer, cloner: cloner, version: version}
}

func (a *MCPAction) Execute(ctx context.Context, _ *cli.Command) error {
	cfg, _ := services.Load()
	if cfg == nil {
		cfg = &domain.Config{}
	}
	server := mcp.New(a.parser, a.replacer, a.cloner, cfg, a.version)
	return server.Serve(ctx)
}
