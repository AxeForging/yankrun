package actions

import (
	"github.com/AxeForging/yankrun/domain"
	"github.com/urfave/cli"
)

func templateSettings(c *cli.Context, cfg *domain.Config) (string, string, string) {
	startDelim := c.String("startDelim")
	endDelim := c.String("endDelim")
	fileSizeLimit := c.String("fileSizeLimit")

	if cfg == nil {
		cfg = &domain.Config{}
	}
	if !c.IsSet("startDelim") && cfg.StartDelim != "" {
		startDelim = cfg.StartDelim
	}
	if !c.IsSet("endDelim") && cfg.EndDelim != "" {
		endDelim = cfg.EndDelim
	}
	if !c.IsSet("fileSizeLimit") && cfg.FileSizeLimit != "" {
		fileSizeLimit = cfg.FileSizeLimit
	}
	if startDelim == "" {
		startDelim = "[["
	}
	if endDelim == "" {
		endDelim = "]]"
	}
	if fileSizeLimit == "" {
		fileSizeLimit = "3 mb"
	}

	return startDelim, endDelim, fileSizeLimit
}
