package actions

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/helpers"
	"github.com/AxeForging/yankrun/services"

	"github.com/urfave/cli"
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

func (a *CloneAction) Execute(c *cli.Context) error {
	repoURL := c.String("repo")
	outputDir := c.String("outputDir")
	verbose := c.Bool("verbose")
	input := c.String("input")
	branch := c.String("branch")
	interactive := c.Bool("interactive")
	processTemplates := c.Bool("processTemplates")
	onlyTemplates := c.Bool("onlyTemplates")
	dryRun := c.Bool("dryRun")
	ignoreFlags := c.StringSlice("ignore")

	if sshKey := c.String("ssh-key"); sshKey != "" {
		a.cloner.SetSSHKeyPath(sshKey)
	}

	// Load defaults from config when flags not provided
	cfg, _ := services.Load()
	if cfg == nil {
		cfg = &domain.Config{}
	}
	startDelim, endDelim, fileSizeLimit := templateSettings(c, cfg)

	if repoURL == "" {
		return fmt.Errorf("--repo is required for clone command")
	}
	if outputDir == "" && !dryRun {
		return fmt.Errorf("--outputDir is required for clone command")
	}

	// Validate flag combination
	if onlyTemplates && !processTemplates {
		return fmt.Errorf("--onlyTemplates requires --processTemplates to be set")
	}

	workDir := outputDir
	if dryRun {
		tmp, err := os.MkdirTemp("", "yankrun-clone-dryrun-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)
		workDir = tmp
	} else {
		if err := a.fs.EnsureDir(outputDir); err != nil {
			return err
		}
	}

	if branch != "" {
		if err := a.cloner.CloneRepositoryBranch(repoURL, branch, workDir); err != nil {
			return err
		}
	} else {
		if err := a.cloner.CloneRepository(repoURL, workDir); err != nil {
			return err
		}
	}

	if dryRun {
		helpers.Log.Info().Msg("Cloned into temporary directory for dry run")
	} else {
		helpers.Log.Info().Msgf("Cloned into %s", outputDir)
	}

	// Parse provided replacements if any
	var provided domain.InputReplacement
	if input != "" {
		var err error
		provided, err = a.parser.Parse(input)
		if err != nil {
			return err
		}
	}

	// Merge ignore patterns from flags and input file
	ignorePatterns := append(ignoreFlags, provided.IgnorePath...)

	// Analyze placeholders in cloned directory
	counts, err := a.replacer.AnalyzeDir(workDir, fileSizeLimit, startDelim, endDelim, onlyTemplates, ignorePatterns)
	if err != nil {
		return err
	}

	// Build value map from provided input
	values := map[string]string{}
	for _, r := range provided.Variables {
		values[r.Key] = r.Value
	}

	// If interactive, prompt for each discovered key
	final := domain.InputReplacement{}
	if len(counts) > 0 {
		keys := make([]string, 0, len(counts))
		for k := range counts {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		helpers.Log.Info().Msg("Discovered placeholders:")
		for _, k := range keys {
			v := values[k]
			if v == "" {
				v = "(unset)"
			}
			fmt.Printf("  %-24s  matches=%-6d  value=%s\n", k, counts[k], v)
		}

		if interactive {
			r := bufio.NewReader(os.Stdin)
			for _, k := range keys {
				def := values[k]
				fmt.Printf("Enter value for %s [%s]: ", k, def)
				s, _ := r.ReadString('\n')
				s = strings.TrimSpace(s)
				if s != "" {
					values[k] = s
				}
			}
			fmt.Println()
		}

		for _, k := range keys {
			if v, ok := values[k]; ok && v != "" {
				final.Variables = append(final.Variables, domain.Replacement{Key: k, Value: v})
			}
		}
	} else {
		// No discovered keys; use provided values directly
		final = provided
	}

	// Dry-run: show summary and exit without writing
	if dryRun {
		totalMatches := 0
		for _, c := range counts {
			totalMatches += c
		}
		helpers.Log.Info().Msgf("Dry run: %d replacements across %d placeholders would be applied. No files modified.", totalMatches, len(final.Variables))
		return nil
	}

	// Skip regular templating if onlyTemplates is set
	if !onlyTemplates {
		if err := a.replacer.ReplaceInDir(workDir, final, fileSizeLimit, startDelim, endDelim, verbose, ignorePatterns); err != nil {
			return err
		}
	}

	// Process .tpl files if requested
	if processTemplates {
		if err := a.replacer.ProcessTemplateFiles(workDir, final, fileSizeLimit, startDelim, endDelim, verbose, ignorePatterns); err != nil {
			return err
		}
		helpers.Log.Info().Msg("Template file processing complete.")
	}

	helpers.Log.Info().Msg("Templating complete.")

	return nil
}
