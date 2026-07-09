package actions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/helpers"
	"github.com/AxeForging/yankrun/internal/schema"
	"github.com/AxeForging/yankrun/internal/ui"
	"github.com/AxeForging/yankrun/internal/workflow"
	"github.com/AxeForging/yankrun/services"
	"github.com/urfave/cli/v3"
)

// ScanAction reports the placeholders discovered in a directory or repo without
// writing anything. It is the read-only entry point agents use to learn what a
// template expects before providing values.
type ScanAction struct {
	fs       services.FileSystem
	parser   services.ReplacementParser
	replacer services.Replacer
	cloner   services.Cloner
}

func NewScanAction(fs services.FileSystem, parser services.ReplacementParser, replacer services.Replacer, cloner services.Cloner) *ScanAction {
	return &ScanAction{fs: fs, parser: parser, replacer: replacer, cloner: cloner}
}

func (a *ScanAction) Execute(_ context.Context, cmd *cli.Command) error {
	jsonOut := cmd.Bool("json")
	if jsonOut {
		// Keep stdout clean for the envelope; logs stay on stderr at warn+.
		helpers.SetupLogger("warn")
	}

	summary, err := a.scan(cmd)

	// In JSON mode every outcome — success or failure — is one envelope so
	// agents never have to parse human text or scrape stderr.
	if jsonOut {
		return schema.Emit(os.Stdout, "scan", summary, err)
	}
	if err != nil {
		return err
	}

	ui.PrintScanSummary(os.Stdout, summary)
	if len(summary.Keys) == 0 {
		fmt.Fprintln(os.Stdout, "No placeholders found.")
	}
	return nil
}

// scan performs the read-only scan and returns its summary. All errors are
// tagged with an exit code so both the JSON envelope and the process exit agree.
func (a *ScanAction) scan(cmd *cli.Command) (workflow.Summary, error) {
	dir := cmd.String("dir")
	repo := cmd.String("repo")

	if dir == "" && repo == "" {
		return workflow.Summary{}, helpers.UsageErr("scan requires --dir or --repo")
	}
	if dir != "" && repo != "" {
		return workflow.Summary{}, helpers.UsageErr("scan accepts either --dir or --repo, not both")
	}

	cfg, _ := services.Load()
	if cfg == nil {
		cfg = &domain.Config{}
	}
	startDelim, endDelim, fileSizeLimit := templateSettings(cmd, cfg)

	if sshKey := cmd.String("ssh-key"); sshKey != "" {
		a.cloner.SetSSHKeyPath(sshKey)
	}

	// Repo scans clone into a throwaway directory that is always cleaned up.
	target := dir
	if repo != "" {
		tmp, err := os.MkdirTemp("", "yankrun-scan-*")
		if err != nil {
			return workflow.Summary{}, err
		}
		defer os.RemoveAll(tmp)
		branch := cmd.String("branch")
		if branch != "" {
			if err := a.cloner.CloneRepositoryBranch(repo, branch, tmp); err != nil {
				return workflow.Summary{}, helpers.GitErr(err)
			}
		} else if err := a.cloner.CloneRepository(repo, tmp); err != nil {
			return workflow.Summary{}, helpers.GitErr(err)
		}
		_ = os.RemoveAll(filepath.Join(tmp, ".git"))
		target = tmp
	}

	var provided domain.InputReplacement
	if input := cmd.String("input"); input != "" {
		parsed, err := a.parser.Parse(input)
		if err != nil {
			return workflow.Summary{}, helpers.ValidationErr("%v", err)
		}
		provided = parsed
	}

	engine := workflow.Engine{Parser: a.parser, Replacer: a.replacer, Cloner: a.cloner}
	settings := workflow.TemplateSettings{
		StartDelim:     startDelim,
		EndDelim:       endDelim,
		FileSizeLimit:  fileSizeLimit,
		OnlyTemplates:  cmd.Bool("onlyTemplates"),
		IgnorePatterns: append(cmd.StringSlice("ignore"), provided.IgnorePath...),
	}

	summary, err := engine.ScanDir(target, settings, provided)
	if err != nil {
		if os.IsNotExist(err) {
			return workflow.Summary{}, helpers.NotFoundErr("%v", err)
		}
		return workflow.Summary{}, err
	}
	return summary, nil
}
