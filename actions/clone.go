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

type CloneAction struct {
	fs       services.FileSystem
	parser   services.ReplacementParser
	replacer services.Replacer
	cloner   services.Cloner
}

func NewCloneAction(fs services.FileSystem, parser services.ReplacementParser, replacer services.Replacer, cloner services.Cloner) *CloneAction {
	return &CloneAction{
		fs:       fs,
		parser:   parser,
		replacer: replacer,
		cloner:   cloner,
	}
}

func (a *CloneAction) Execute(_ context.Context, cmd *cli.Command) error {
	jsonOut := cmd.Bool("json")
	if jsonOut {
		helpers.SetupLogger("warn")
	}

	result, manifest, dryRun, err := a.run(cmd)

	if jsonOut {
		return schema.Emit(os.Stdout, "clone", result, err)
	}
	if err != nil {
		return err
	}
	printApply(result, manifest, dryRun)
	return nil
}

func (a *CloneAction) run(cmd *cli.Command) (workflow.ApplyResult, *domain.Manifest, bool, error) {
	repoURL := cmd.String("repo")
	outputDir := cmd.String("outputDir")
	branch := cmd.String("branch")
	dryRun := cmd.Bool("dryRun")

	if repoURL == "" {
		return workflow.ApplyResult{}, nil, false, helpers.UsageErr("--repo is required for clone command")
	}
	if outputDir == "" && !dryRun {
		return workflow.ApplyResult{}, nil, false, helpers.UsageErr("--outputDir is required for clone command")
	}
	if cmd.Bool("onlyTemplates") && !cmd.Bool("processTemplates") {
		return workflow.ApplyResult{}, nil, false, helpers.UsageErr("--onlyTemplates requires --processTemplates to be set")
	}
	if sshKey := cmd.String("ssh-key"); sshKey != "" {
		a.cloner.SetSSHKeyPath(sshKey)
	}

	cfg, _ := services.Load()
	if cfg == nil {
		cfg = &domain.Config{}
	}
	startDelim, endDelim, fileSizeLimit := templateSettings(cmd, cfg)

	// Clone into the output dir, or a throwaway temp dir on dry runs.
	workDir := outputDir
	if dryRun {
		tmp, err := os.MkdirTemp("", "yankrun-clone-dryrun-*")
		if err != nil {
			return workflow.ApplyResult{}, nil, false, err
		}
		defer os.RemoveAll(tmp)
		workDir = tmp
	} else if err := a.fs.EnsureDir(outputDir); err != nil {
		return workflow.ApplyResult{}, nil, false, err
	}

	if branch != "" {
		if err := a.cloner.CloneRepositoryBranch(repoURL, branch, workDir); err != nil {
			return workflow.ApplyResult{}, nil, false, helpers.GitErr(err)
		}
	} else if err := a.cloner.CloneRepository(repoURL, workDir); err != nil {
		return workflow.ApplyResult{}, nil, false, helpers.GitErr(err)
	}
	if dryRun {
		helpers.Log.Info().Msg("Cloned into a temporary directory for dry run")
	} else {
		helpers.Log.Info().Msgf("Cloned into %s", outputDir)
	}

	provided, err := parseInput(a.parser, cmd.String("input"))
	if err != nil {
		return workflow.ApplyResult{}, nil, dryRun, err
	}

	engine := workflow.Engine{Parser: a.parser, Replacer: a.replacer, Cloner: a.cloner}
	result, manifest, err := runApply(engine, applyOptions{
		dir:      workDir,
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
