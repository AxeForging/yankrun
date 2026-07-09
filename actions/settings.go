package actions

import (
	"github.com/AxeForging/yankrun/domain"
	"github.com/urfave/cli/v3"
)

func templateSettings(cmd *cli.Command, cfg *domain.Config) (string, string, string) {
	startDelim := cmd.String("startDelim")
	endDelim := cmd.String("endDelim")
	fileSizeLimit := cmd.String("fileSizeLimit")

	if cfg == nil {
		cfg = &domain.Config{}
	}
	if !cmd.IsSet("startDelim") && cfg.StartDelim != "" {
		startDelim = cfg.StartDelim
	}
	if !cmd.IsSet("endDelim") && cfg.EndDelim != "" {
		endDelim = cfg.EndDelim
	}
	if !cmd.IsSet("fileSizeLimit") && cfg.FileSizeLimit != "" {
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
