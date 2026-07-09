// Package mcp exposes yankrun's templating engine over the Model Context
// Protocol (stdio), so agents like Claude Code can scan, preview, and apply
// templates natively. Every tool is a thin wrapper over workflow.Engine and
// returns the same JSON-tagged types as the CLI's --json output.
package mcp

import (
	"context"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/internal/workflow"
	"github.com/AxeForging/yankrun/services"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server holds the dependencies the MCP tools operate over.
type Server struct {
	engine workflow.Engine
	cfg    *domain.Config
	// version is reported in the MCP initialize handshake.
	version string
}

// New builds an MCP server from the shared services and config.
func New(parser services.ReplacementParser, replacer services.Replacer, cloner services.Cloner, cfg *domain.Config, version string) *Server {
	if cfg == nil {
		cfg = &domain.Config{}
	}
	return &Server{
		engine:  workflow.Engine{Parser: parser, Replacer: replacer, Cloner: cloner},
		cfg:     cfg,
		version: version,
	}
}

// Serve registers the tools and serves MCP over stdio until ctx is cancelled or
// the client disconnects.
func (s *Server) Serve(ctx context.Context) error {
	return s.sdkServer().Run(ctx, &mcpsdk.StdioTransport{})
}

// sdkServer builds the underlying SDK server with all tools registered. Tests
// connect it to an in-memory transport.
func (s *Server) sdkServer() *mcpsdk.Server {
	srv := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "yankrun",
		Version: s.version,
	}, nil)
	s.registerTools(srv)
	return srv
}

// settings builds template settings from tool input, falling back to config and
// then built-in defaults for the delimiters and size limit.
func (s *Server) settings(start, end string, ignore []string, onlyTemplates, processTemplates bool) workflow.TemplateSettings {
	if start == "" {
		start = s.cfg.StartDelim
	}
	if end == "" {
		end = s.cfg.EndDelim
	}
	if start == "" {
		start = "[["
	}
	if end == "" {
		end = "]]"
	}
	limit := s.cfg.FileSizeLimit
	if limit == "" {
		limit = "3 mb"
	}
	return workflow.TemplateSettings{
		StartDelim:       start,
		EndDelim:         end,
		FileSizeLimit:    limit,
		IgnorePatterns:   ignore,
		OnlyTemplates:    onlyTemplates,
		ProcessTemplates: processTemplates,
	}
}
