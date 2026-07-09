package ui

import (
	"fmt"
	"io"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/internal/workflow"
	"github.com/charmbracelet/lipgloss"
)

// PrintScanSummary renders the discovered placeholders of a scan: each key, its
// match count, and the resolved value (or "(unset)"). This is the human twin of
// the --json scan envelope.
func PrintScanSummary(w io.Writer, s workflow.Summary) {
	if len(s.Keys) == 0 {
		return
	}
	fmt.Fprintln(w, SectionHeader("Discovered placeholders"))
	width := longestKey(s.Keys)
	for _, k := range s.Keys {
		value := s.Values[k]
		valStyle := lipgloss.NewStyle()
		if value == "" {
			value = "(unset)"
			valStyle = Muted
		}
		meta := Count.Render(fmt.Sprintf("%d %s", s.Counts[k], matchWord(s.Counts[k])))
		fmt.Fprintln(w, keyValue(k, meta+"  "+valStyle.Render(value), width))
	}
}

// PrintApplyResult renders the outcome of an apply (or dry run): totals plus a
// terminal status line. Callers print the placeholder summary separately.
func PrintApplyResult(w io.Writer, r workflow.ApplyResult, dryRun bool) {
	totals := fmt.Sprintf("%d replacement%s across %d placeholder%s",
		r.TotalMatches, plural(r.TotalMatches), r.Placeholders, plural(r.Placeholders))
	switch {
	case dryRun || r.ForcedDryRun:
		writeLines(w, StatusInfo("Dry run: "+totals+". No files modified."))
	case r.Applied:
		writeLines(w, StatusOK("Applied "+totals+"."))
	default:
		writeLines(w, StatusInfo("Nothing to apply."))
	}
}

// PrintHints renders a manifest's post-generate hints as a "Next steps"
// section. It is a no-op when the manifest has none.
func PrintHints(w io.Writer, manifest *domain.Manifest) {
	if manifest == nil || len(manifest.PostGenerate.Hints) == 0 {
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, SectionHeader("Next steps"))
	for _, hint := range manifest.PostGenerate.Hints {
		fmt.Fprintln(w, "  "+Accent.Render(GlyphDot)+" "+hint)
	}
}

// PrintDiff renders a unified diff with colored +/- lines.
func PrintDiff(w io.Writer, path, diff string) {
	if diff == "" {
		return
	}
	fmt.Fprintln(w, Accent.Render(GlyphArrow+" ")+Bold.Render(path))
	for _, line := range splitLines(diff) {
		switch {
		case len(line) > 0 && line[0] == '+':
			fmt.Fprintln(w, Success.Render(line))
		case len(line) > 0 && line[0] == '-':
			fmt.Fprintln(w, Error.Render(line))
		default:
			fmt.Fprintln(w, Muted.Render(line))
		}
	}
}

func longestKey(keys []string) int {
	width := 0
	for _, k := range keys {
		if lipgloss.Width(k) > width {
			width = lipgloss.Width(k)
		}
	}
	return width
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func matchWord(n int) string {
	if n == 1 {
		return "match"
	}
	return "matches"
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
