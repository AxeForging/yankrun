package tui

import (
	"fmt"

	"github.com/AxeForging/yankrun/internal/workflow"
	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	switch m.state {
	case stateScanning:
		return m.frame(m.spinner.View() + " Scanning " + keyStyle.Render(m.dir) + "…")

	case stateForm:
		body := lipgloss.JoinVertical(lipgloss.Left,
			m.placeholderLine(),
			"",
			m.form.View(),
		)
		return m.frame(body)

	case statePreviewLoading:
		return m.frame(m.spinner.View() + " Computing preview…")

	case statePreview:
		title := titleStyle.Render("▸ Preview")
		hint := subtleStyle.Render(fmt.Sprintf("%d replacement%s across %d placeholder%s",
			m.result.TotalMatches, plural(m.result.TotalMatches), m.result.Placeholders, plural(m.result.Placeholders)))
		body := lipgloss.JoinVertical(lipgloss.Left, title+"  "+hint, "", m.viewport.View())
		return m.frameHelp(body, "a apply · e edit values · ↑/↓ scroll · q quit")

	case stateApplying:
		return m.frame(m.spinner.View() + " Applying…")

	case stateDone:
		var status string
		if m.result.Applied {
			status = okStyle.Render("✓ Applied " + summaryTotals(m.result))
		} else if m.dryRun {
			status = subtleStyle.Render("• Dry run: " + summaryTotals(m.result) + ". No files modified.")
		} else {
			status = subtleStyle.Render("• Nothing to apply.")
		}
		return m.frameHelp(status, "q quit")

	case stateEmpty:
		return m.frameHelp(subtleStyle.Render("No placeholders found in "+m.dir+"."), "q quit")

	case stateError:
		return m.frameHelp(errStyle.Render("✗ "+m.err.Error()), "q quit")
	}
	return ""
}

// frame wraps body with the header bar.
func (m model) frame(body string) string {
	return m.header() + "\n\n" + body
}

// frameHelp wraps body with the header bar and a help footer.
func (m model) frameHelp(body, help string) string {
	return m.header() + "\n\n" + body + "\n\n" + helpBar.Render(help)
}

func (m model) header() string {
	mode := ""
	if m.dryRun {
		mode = subtleStyle.Render("  (dry run)")
	}
	return headerBar.Render(mark+" yankrun") + subtleStyle.Render(" · templating workbench") + mode
}

func (m model) placeholderLine() string {
	return subtleStyle.Render(fmt.Sprintf("%d placeholder%s across %d file%s in %s",
		len(m.summary.Keys), plural(len(m.summary.Keys)),
		len(m.summary.Files), plural(len(m.summary.Files)), m.dir))
}

func summaryTotals(r workflow.ApplyResult) string {
	return fmt.Sprintf("%d replacement%s across %d placeholder%s",
		r.TotalMatches, plural(r.TotalMatches), r.Placeholders, plural(r.Placeholders))
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
