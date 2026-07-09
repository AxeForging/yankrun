package actions

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/helpers"
	"github.com/AxeForging/yankrun/internal/schema"
	"github.com/AxeForging/yankrun/internal/workflow"
	"github.com/AxeForging/yankrun/services"
	"github.com/urfave/cli/v3"
)

type GenerateAction struct {
	fs       services.FileSystem
	cloner   services.Cloner
	parser   services.ReplacementParser
	replacer services.Replacer
}

func NewGenerateAction(fs services.FileSystem, cloner services.Cloner, parser services.ReplacementParser, replacer services.Replacer) *GenerateAction {
	return &GenerateAction{fs: fs, cloner: cloner, parser: parser, replacer: replacer}
}

// Execute: choose template repo/branch, clone, remove .git, then resolve and
// apply replacements. Supports --json for a machine-readable envelope.
func (a *GenerateAction) Execute(_ context.Context, cmd *cli.Command) error {
	jsonOut := cmd.Bool("json")
	if jsonOut {
		helpers.SetupLogger("warn")
	}

	result, manifest, dryRun, err := a.run(cmd)

	if jsonOut {
		return schema.Emit(os.Stdout, "generate", result, err)
	}
	if err != nil {
		return err
	}
	printApply(result, manifest, dryRun)
	return nil
}

func (a *GenerateAction) run(cmd *cli.Command) (workflow.ApplyResult, *domain.Manifest, bool, error) {
	interactivePrompt := cmd.Bool("interactive")
	jsonOut := cmd.Bool("json")
	input := cmd.String("input")
	outputDir := cmd.String("outputDir")
	templateFilter := cmd.String("template")
	branchFlag := cmd.String("branch")
	dryRun := cmd.Bool("dryRun")
	noCache := cmd.Bool("noCache")

	// interactiveAllowed gates every prompt so --json, --yes, and non-TTY runs
	// never block waiting for input.
	interactiveAllowed := interactivePrompt && !jsonOut && !cmd.Bool("yes") && helpers.IsInteractive()

	if cmd.Bool("onlyTemplates") && !cmd.Bool("processTemplates") {
		return workflow.ApplyResult{}, nil, dryRun, helpers.UsageErr("--onlyTemplates requires --processTemplates to be set")
	}
	if sshKey := cmd.String("ssh-key"); sshKey != "" {
		a.cloner.SetSSHKeyPath(sshKey)
	}

	cfg, err := services.Load()
	if err != nil {
		return workflow.ApplyResult{}, nil, dryRun, fmt.Errorf("failed to load config: %w", err)
	}
	if len(cfg.Templates) == 0 && cfg.GitHub.User == "" && len(cfg.GitHub.Orgs) == 0 && templateFilter == "" && interactiveAllowed {
		// Minimal discovery setup, only when we can actually prompt.
		r := bufio.NewReader(os.Stdin)
		fmt.Println("No templates configured. Let's set where to search:")
		u := strings.TrimSpace(promptString(r, "GitHub user (optional, Enter to skip)", ""))
		orgsCSV := strings.TrimSpace(promptString(r, "GitHub orgs (comma-separated, optional)", ""))
		var orgs []string
		for _, p := range strings.Split(orgsCSV, ",") {
			if s := strings.TrimSpace(p); s != "" {
				orgs = append(orgs, s)
			}
		}
		cfg.GitHub.User = u
		cfg.GitHub.Orgs = orgs
		_ = services.Save(cfg)
	}

	startDelim, endDelim, fileSizeLimit := templateSettings(cmd, cfg)

	cache, _ := services.LoadCache()
	cacheUpdated := false

	repos := cfg.Templates
	if templateFilter != "" && (strings.Contains(templateFilter, "://") || strings.HasPrefix(templateFilter, "git@")) {
		repos = append(repos, domain.TemplateRepo{Name: templateFilter, URL: templateFilter, DefaultBranch: "main"})
	}
	if cfg.GitHub.User != "" || len(cfg.GitHub.Orgs) > 0 {
		configSHA := services.GitHubConfigSHA(cfg.GitHub)
		if !noCache && configSHA == cache.GitHubConfigSHA && len(cache.GitHubRepos) > 0 {
			repos = append(repos, cache.GitHubRepos...)
			helpers.Log.Debug().Msg("Using cached GitHub repos")
		} else {
			ghClient := services.NewGitHubClient()
			found, err := ghClient.ListRepos(context.Background(), cfg.GitHub)
			if err != nil {
				helpers.Log.Warn().Err(err).Msg("Failed to discover GitHub repos")
			}
			var ghRepos []domain.TemplateRepo
			for _, fr := range found {
				tr := domain.TemplateRepo{Name: fr.FullName, URL: fr.SSHURL, Description: fr.Description, DefaultBranch: fr.DefaultBranch}
				repos = append(repos, tr)
				ghRepos = append(ghRepos, tr)
			}
			cache.GitHubConfigSHA = configSHA
			cache.GitHubRepos = ghRepos
			cacheUpdated = true
		}
	}
	if len(repos) == 0 {
		return workflow.ApplyResult{}, nil, dryRun, helpers.NotFoundErr("no templates configured or found")
	}

	chosen, err := a.selectTemplate(repos, templateFilter, interactivePrompt, interactiveAllowed)
	if err != nil {
		return workflow.ApplyResult{}, nil, dryRun, err
	}

	br := a.selectBranch(chosen, branchFlag, interactivePrompt, interactiveAllowed)

	if outputDir == "" && !dryRun {
		if interactiveAllowed {
			r := bufio.NewReader(os.Stdin)
			fmt.Printf("Output directory [./new-project]: ")
			out, _ := r.ReadString('\n')
			if out = strings.TrimSpace(out); out == "" {
				out = "./new-project"
			}
			outputDir = out
		} else {
			return workflow.ApplyResult{}, nil, dryRun, helpers.UsageErr("--outputDir is required for non-interactive generate")
		}
	}

	workDir := outputDir
	if dryRun {
		tmp, err := os.MkdirTemp("", "yankrun-generate-dryrun-*")
		if err != nil {
			return workflow.ApplyResult{}, nil, dryRun, err
		}
		defer os.RemoveAll(tmp)
		workDir = tmp
	} else if err := a.fs.EnsureDir(outputDir); err != nil {
		return workflow.ApplyResult{}, nil, dryRun, err
	}

	if err := a.cloner.CloneRepositoryBranch(chosen.URL, br, workDir); err != nil {
		return workflow.ApplyResult{}, nil, dryRun, helpers.GitErr(err)
	}
	if dryRun {
		helpers.Log.Info().Msgf("Cloned %s@%s into a temporary directory for dry run", chosen.Name, br)
	} else {
		helpers.Log.Info().Msgf("Cloned %s@%s into %s", chosen.Name, br, outputDir)
	}

	headSHA, _ := services.HeadSHA(workDir)
	if err := os.RemoveAll(filepath.Join(workDir, ".git")); err != nil {
		return workflow.ApplyResult{}, nil, dryRun, fmt.Errorf("failed to remove .git: %w", err)
	}
	helpers.Log.Info().Msg("Removed .git directory (new repo initialized)")

	provided, err := parseInput(a.parser, input)
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
		interactive: interactiveAllowed,
		dryRun:      dryRun,
	})
	if err != nil {
		return workflow.ApplyResult{}, manifest, dryRun, err
	}

	// Refresh the discovered-variables cache from the scan counts.
	if headSHA != "" && len(result.Summary.Counts) > 0 {
		services.UpdateVars(cache, chosen.URL, br, headSHA, result.Summary.Counts)
		cacheUpdated = true
	}
	if cacheUpdated {
		_ = services.SaveCache(cache)
	}
	return result, manifest, dryRun, nil
}

// selectTemplate resolves the template to use, prompting only when allowed.
func (a *GenerateAction) selectTemplate(repos []domain.TemplateRepo, filter string, interactivePrompt, interactiveAllowed bool) (domain.TemplateRepo, error) {
	var filtered []domain.TemplateRepo
	if filter != "" {
		for _, t := range repos {
			if strings.Contains(strings.ToLower(t.Name), strings.ToLower(filter)) || strings.Contains(strings.ToLower(t.URL), strings.ToLower(filter)) {
				filtered = append(filtered, t)
			}
		}
	} else {
		filtered = repos
	}
	if len(filtered) == 0 {
		return domain.TemplateRepo{}, helpers.NotFoundErr("no templates matched filter")
	}

	switch {
	case filter != "" && !interactivePrompt:
		return filtered[0], nil
	case interactiveAllowed:
		return a.promptTemplate(repos), nil
	case len(filtered) == 1:
		return filtered[0], nil
	default:
		return domain.TemplateRepo{}, helpers.UsageErr("--template is required to select a template non-interactively")
	}
}

// promptTemplate runs the interactive template picker.
func (a *GenerateAction) promptTemplate(repos []domain.TemplateRepo) domain.TemplateRepo {
	r := bufio.NewReader(os.Stdin)
	helpers.Log.Info().Msg("Available templates:")
	for i, t := range repos {
		fmt.Printf("  [%d] %s  (%s)\n", i+1, t.Name, t.URL)
	}
	fmt.Printf("Filter templates by substring (press Enter to skip): ")
	filter, _ := r.ReadString('\n')
	filter = strings.TrimSpace(filter)
	filtered := repos
	if filter != "" {
		var match []domain.TemplateRepo
		for _, t := range repos {
			if strings.Contains(strings.ToLower(t.Name), strings.ToLower(filter)) || strings.Contains(strings.ToLower(t.URL), strings.ToLower(filter)) {
				match = append(match, t)
			}
		}
		if len(match) > 0 {
			filtered = match
		}
	}
	for i, t := range filtered {
		fmt.Printf("  [%d] %s  (%s)\n", i+1, t.Name, t.URL)
	}
	fmt.Printf("Select template [1-%d]: ", len(filtered))
	selStr, _ := r.ReadString('\n')
	idx := 0
	if selStr = strings.TrimSpace(selStr); selStr != "" {
		fmt.Sscanf(selStr, "%d", &idx)
		idx--
	}
	if idx < 0 || idx >= len(filtered) {
		idx = 0
	}
	return filtered[idx]
}

// selectBranch resolves the branch, prompting only when allowed.
func (a *GenerateAction) selectBranch(chosen domain.TemplateRepo, branchFlag string, interactivePrompt, interactiveAllowed bool) string {
	if !interactivePrompt || !interactiveAllowed {
		switch {
		case branchFlag != "":
			return branchFlag
		case chosen.DefaultBranch != "":
			return chosen.DefaultBranch
		default:
			return "main"
		}
	}

	branches, _ := a.cloner.ListRemoteBranches(chosen.URL)
	if len(branches) == 0 && chosen.DefaultBranch != "" {
		branches = []string{chosen.DefaultBranch}
	}
	r := bufio.NewReader(os.Stdin)
	fmt.Printf("Type to filter branches (Enter to accept default [%s]): ", chosen.DefaultBranch)
	branchFilter, _ := r.ReadString('\n')
	branchFilter = strings.TrimSpace(branchFilter)
	candidates := branches
	if branchFilter != "" {
		var match []string
		for _, b := range branches {
			if strings.Contains(strings.ToLower(b), strings.ToLower(branchFilter)) {
				match = append(match, b)
			}
		}
		if len(match) > 0 {
			candidates = match
		}
	}
	fmt.Println("Available branches:")
	for i, b := range candidates {
		fmt.Printf("  [%d] %s\n", i+1, b)
	}
	fmt.Printf("Select branch [1-%d] (Enter for default): ", len(candidates))
	pick, _ := r.ReadString('\n')
	if pick = strings.TrimSpace(pick); pick != "" {
		var idx int
		if _, err := fmt.Sscanf(pick, "%d", &idx); err == nil && idx >= 1 && idx <= len(candidates) {
			return candidates[idx-1]
		}
	}
	if chosen.DefaultBranch != "" {
		return chosen.DefaultBranch
	}
	return "main"
}

// promptString reads a line with a label and returns the trimmed value or default when empty.
func promptString(r *bufio.Reader, label string, def string) string {
	if def == "" {
		fmt.Printf("%s: ", label)
	} else {
		fmt.Printf("%s [%s]: ", label, def)
	}
	if v, err := r.ReadString('\n'); err == nil {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return def
}
