package tui

import (
	"github.com/AxeForging/yankrun/internal/ui"
	"github.com/charmbracelet/lipgloss"
)

// Colors are pulled from the shared ui theme so the TUI matches the CLI.
var accentC, mutedC, successC, errorC, _, _ = ui.Colors()

var (
	titleStyle  = lipgloss.NewStyle().Foreground(accentC).Bold(true)
	subtleStyle = lipgloss.NewStyle().Foreground(mutedC)
	okStyle     = lipgloss.NewStyle().Foreground(successC)
	errStyle    = lipgloss.NewStyle().Foreground(errorC)
	keyStyle    = lipgloss.NewStyle().Bold(true)

	headerBar = lipgloss.NewStyle().
			Foreground(accentC).
			Bold(true).
			Padding(0, 1)

	helpBar = lipgloss.NewStyle().
		Foreground(mutedC).
		Padding(0, 1)
)

const mark = "⟦y⟧"
