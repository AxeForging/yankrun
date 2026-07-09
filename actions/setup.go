package actions

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/helpers"
	"github.com/AxeForging/yankrun/internal/ui"
	"github.com/AxeForging/yankrun/services"
	"github.com/charmbracelet/huh"
)

// RunSetup configures defaults (~/.yankrun/config.yaml). When show is true it
// prints the current config; when reset is true it deletes the config file.
func RunSetup(show, reset bool) error {
	if reset {
		if err := services.Reset(); err != nil {
			return errors.New("failed to delete config: " + err.Error())
		}
		helpers.Log.Info().Msg("Configuration removed ✔")
		return nil
	}

	cfg, err := services.Load()
	if err != nil {
		// proceed with empty config if file doesn't exist yet
		cfg = &domain.Config{}
	}

	if show {
		printConfig(cfg)
		return nil
	}

	if !helpers.IsInteractive() {
		return helpers.UsageErr("setup requires an interactive terminal; use --show or --reset, or edit ~/.yankrun/config.yaml directly")
	}

	if cfg.StartDelim == "" {
		cfg.StartDelim = "[["
	}
	if cfg.EndDelim == "" {
		cfg.EndDelim = "]]"
	}
	if cfg.FileSizeLimit == "" {
		cfg.FileSizeLimit = "3 mb"
	}

	// Core defaults in one grouped form.
	form := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Template start delimiter").Description("e.g. [[").Value(&cfg.StartDelim),
		huh.NewInput().Title("Template end delimiter").Description("e.g. ]]").Value(&cfg.EndDelim),
		huh.NewInput().Title("File size limit").Description("skip files larger than this, e.g. 3 mb").Value(&cfg.FileSizeLimit),
	)).WithTheme(ui.Theme())
	if err := form.Run(); err != nil {
		return promptError(err)
	}

	if err := addTemplates(cfg); err != nil {
		return err
	}
	if err := configureGitHub(cfg); err != nil {
		return err
	}

	if err := services.Save(cfg); err != nil {
		return errors.New("failed to save config: " + err.Error())
	}
	helpers.Log.Info().Msg("Configuration saved ✔")
	return nil
}

// addTemplates loops, appending template repos until the user declines.
func addTemplates(cfg *domain.Config) error {
	for {
		add, err := ui.Confirm("Add a template repo?", false)
		if err != nil {
			return promptError(err)
		}
		if !add {
			return nil
		}

		t := domain.TemplateRepo{DefaultBranch: "main"}
		form := huh.NewForm(huh.NewGroup(
			huh.NewInput().Title("Template name").Description("label, e.g. 'Go App' or 'org/repo'").Value(&t.Name),
			huh.NewInput().Title("Template git URL").Description("SSH or HTTPS").Value(&t.URL),
			huh.NewInput().Title("Description").Description("optional").Value(&t.Description),
			huh.NewInput().Title("Default branch").Value(&t.DefaultBranch),
		)).WithTheme(ui.Theme())
		if err := form.Run(); err != nil {
			return promptError(err)
		}
		if strings.TrimSpace(t.URL) != "" {
			if t.DefaultBranch == "" {
				t.DefaultBranch = "main"
			}
			cfg.Templates = append(cfg.Templates, t)
		}
	}
}

// configureGitHub optionally sets up GitHub-based template discovery.
func configureGitHub(cfg *domain.Config) error {
	want, err := ui.Confirm("Configure GitHub discovery?", false)
	if err != nil {
		return promptError(err)
	}
	if !want {
		return nil
	}

	orgsCSV := strings.Join(cfg.GitHub.Orgs, ",")
	includePrivate := cfg.GitHub.IncludePrivate
	form := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("GitHub user").Description("leave empty to skip").Value(&cfg.GitHub.User),
		huh.NewInput().Title("GitHub orgs").Description("comma-separated").Value(&orgsCSV),
		huh.NewInput().Title("Filter by topic").Description("optional").Value(&cfg.GitHub.Topic),
		huh.NewInput().Title("Filter by name prefix").Description("optional").Value(&cfg.GitHub.Prefix),
		huh.NewConfirm().Title("Include private repos?").Value(&includePrivate),
		huh.NewInput().Title("GitHub token").Description("optional — higher rate limits / private repos").Value(&cfg.GitHub.Token),
	)).WithTheme(ui.Theme())
	if err := form.Run(); err != nil {
		return promptError(err)
	}

	cfg.GitHub.IncludePrivate = includePrivate
	cfg.GitHub.Orgs = cfg.GitHub.Orgs[:0]
	for _, p := range strings.Split(orgsCSV, ",") {
		if s := strings.TrimSpace(p); s != "" {
			cfg.GitHub.Orgs = append(cfg.GitHub.Orgs, s)
		}
	}
	return nil
}

// printConfig displays the current configuration.
func printConfig(cfg *domain.Config) {
	helpers.Log.Info().Msg("Current configuration:")
	fmt.Printf("\n  start_delim:     %q\n  end_delim:       %q\n  file_size_limit: %s\n", cfg.StartDelim, cfg.EndDelim, cfg.FileSizeLimit)
	fmt.Printf("  templates:       %d configured\n", len(cfg.Templates))
	for i, t := range cfg.Templates {
		fmt.Printf("    - [%d] %s (%s) default_branch=%s\n", i+1, t.Name, t.URL, t.DefaultBranch)
	}
	if cfg.GitHub.User != "" || len(cfg.GitHub.Orgs) > 0 || cfg.GitHub.Topic != "" || cfg.GitHub.Prefix != "" || cfg.GitHub.IncludePrivate {
		fmt.Printf("  github.user:     %s\n", cfg.GitHub.User)
		fmt.Printf("  github.orgs:     %s\n", strings.Join(cfg.GitHub.Orgs, ", "))
		fmt.Printf("  github.topic:    %s\n", cfg.GitHub.Topic)
		fmt.Printf("  github.prefix:   %s\n", cfg.GitHub.Prefix)
		fmt.Printf("  github.private:  %t\n", cfg.GitHub.IncludePrivate)
	}
	fmt.Println()
}

// promptError maps a huh abort to a cancelled exit code.
func promptError(err error) error {
	if errors.Is(err, huh.ErrUserAborted) {
		return helpers.CancelledErr("cancelled")
	}
	return err
}
