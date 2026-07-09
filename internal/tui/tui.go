// Package tui implements yankrun's full-screen interactive templating
// workbench: the terminal twin of `yankrun serve`. It scans a directory, lets
// the user fill placeholder values (manifest-aware), previews the exact diffs,
// and applies on confirmation. All data operations go through workflow.Engine.
package tui

import (
	"io"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/internal/workflow"
	"github.com/AxeForging/yankrun/services"
	tea "github.com/charmbracelet/bubbletea"
)

type Options struct {
	Dir              string
	StartDelim       string
	EndDelim         string
	FileSizeLimit    string
	IgnorePatterns   []string
	ProcessTemplates bool
	OnlyTemplates    bool
	DryRun           bool
	Verbose          bool
	Provided         domain.InputReplacement
	Replacer         services.Replacer
	// In/Out override the program's I/O; used by tests. Production leaves them
	// nil so bubbletea uses the real terminal.
	In  io.Reader
	Out io.Writer
}

// Run starts the interactive workbench and blocks until the user exits.
func Run(opts Options) error {
	if opts.StartDelim == "" {
		opts.StartDelim = "[["
	}
	if opts.EndDelim == "" {
		opts.EndDelim = "]]"
	}
	if opts.FileSizeLimit == "" {
		opts.FileSizeLimit = "3 mb"
	}

	engine := workflow.Engine{Replacer: opts.Replacer}
	settings := workflow.TemplateSettings{
		StartDelim:       opts.StartDelim,
		EndDelim:         opts.EndDelim,
		FileSizeLimit:    opts.FileSizeLimit,
		ProcessTemplates: opts.ProcessTemplates,
		OnlyTemplates:    opts.OnlyTemplates,
		Verbose:          opts.Verbose,
		IgnorePatterns:   opts.IgnorePatterns,
	}

	m := newModel(engine, opts.Dir, settings, opts.Provided, opts.DryRun)

	var progOpts []tea.ProgramOption
	if opts.In != nil {
		progOpts = append(progOpts, tea.WithInput(opts.In))
	}
	if opts.Out != nil {
		progOpts = append(progOpts, tea.WithOutput(opts.Out))
	}
	_, err := tea.NewProgram(m, progOpts...).Run()
	return err
}
