package tui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/helpers"
	"github.com/AxeForging/yankrun/internal/workflow"
	"github.com/AxeForging/yankrun/services"
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
	In               io.Reader
	Out              io.Writer
}

func Run(opts Options) error {
	if opts.In == nil {
		opts.In = os.Stdin
	}
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.StartDelim == "" {
		opts.StartDelim = "[["
	}
	if opts.EndDelim == "" {
		opts.EndDelim = "]]"
	}
	if opts.FileSizeLimit == "" {
		opts.FileSizeLimit = "3 mb"
	}

	fmt.Fprint(opts.Out, "\033[2J\033[H")
	fmt.Fprintln(opts.Out, "YankRun TUI")
	fmt.Fprintln(opts.Out, "Scanning template surface")
	for i := 0; i < 3; i++ {
		fmt.Fprint(opts.Out, ".")
		time.Sleep(80 * time.Millisecond)
	}
	fmt.Fprintln(opts.Out)

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
	summary, err := engine.ScanDir(opts.Dir, settings, opts.Provided)
	if err != nil {
		return err
	}
	if len(summary.Counts) == 0 {
		helpers.Log.Info().Msg("No placeholders found.")
		return nil
	}

	keys := summary.Keys
	values := summary.Values
	reader := bufio.NewReader(opts.In)

	fmt.Fprintln(opts.Out)
	fmt.Fprintln(opts.Out, "Discovered placeholders")
	for _, k := range keys {
		def := values[k]
		if def == "" {
			def = "(unset)"
		}
		fmt.Fprintf(opts.Out, "  %-24s matches=%-6d value=%s\n", k, summary.Counts[k], def)
	}
	fmt.Fprintln(opts.Out)

	for _, k := range keys {
		fmt.Fprintf(opts.Out, "Value for %s [%s]: ", k, values[k])
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)
		if answer != "" {
			values[k] = answer
		}
	}

	final := workflow.BuildFinal(keys, values)
	if len(final.Variables) == 0 {
		helpers.Log.Info().Msg("No values provided; nothing to replace.")
		return nil
	}

	total := workflow.ReplacementMatchCount(keys, summary.Counts, values)
	fmt.Fprintln(opts.Out)
	fmt.Fprintf(opts.Out, "Preview: %d replacements across %d placeholders.\n", total, len(final.Variables))
	if opts.DryRun {
		helpers.Log.Info().Msg("Dry run enabled; no files modified.")
		return nil
	}

	fmt.Fprint(opts.Out, "Apply these changes? Type yes to continue: ")
	answer, _ := reader.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(answer)) != "yes" {
		helpers.Log.Info().Msg("Cancelled; no files modified.")
		return nil
	}

	if err := engine.ApplyFinal(opts.Dir, settings, final); err != nil {
		return err
	}

	helpers.Log.Info().Msg("Templating complete.")
	return nil
}
