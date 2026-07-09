// Package ui renders yankrun's human-facing terminal output: the CLI banner,
// summaries, status lines, help, and the styles shared with the interactive
// prompts and the Bubble Tea TUI. It translates the AxeForge brand (forge
// flavor) into terminal colors that degrade gracefully from truecolor to
// 16-color terminals, and falls back to plain text when stdout is not a TTY or
// NO_COLOR is set (lipgloss handles that via the active color profile).
package ui

import "github.com/charmbracelet/lipgloss"

// Brand palette. Accent is the forge flavor (#ff4e00) with 256- and 16-color
// fallbacks so it survives on lesser terminals. Status colors reuse the brand
// kit's terminal-dot semantics (green/red) plus amber/cyan for warn/info.
var (
	accentColor  = lipgloss.CompleteColor{TrueColor: "#ff4e00", ANSI256: "202", ANSI: "9"}
	mutedColor   = lipgloss.CompleteColor{TrueColor: "#94a3b8", ANSI256: "245", ANSI: "8"}
	successColor = lipgloss.CompleteColor{TrueColor: "#27c93f", ANSI256: "76", ANSI: "2"}
	errorColor   = lipgloss.CompleteColor{TrueColor: "#ff5f56", ANSI256: "203", ANSI: "1"}
	warnColor    = lipgloss.CompleteColor{TrueColor: "#ffb000", ANSI256: "214", ANSI: "3"}
	infoColor    = lipgloss.CompleteColor{TrueColor: "#22d3ee", ANSI256: "45", ANSI: "6"}
)

// Shared styles. Kept as package vars so the CLI renderers, huh theme, and TUI
// all draw from one source of truth.
var (
	Accent  = lipgloss.NewStyle().Foreground(accentColor)
	Muted   = lipgloss.NewStyle().Foreground(mutedColor)
	Bold    = lipgloss.NewStyle().Bold(true)
	Success = lipgloss.NewStyle().Foreground(successColor)
	Error   = lipgloss.NewStyle().Foreground(errorColor)
	Warn    = lipgloss.NewStyle().Foreground(warnColor)
	Info    = lipgloss.NewStyle().Foreground(infoColor)

	// Title is the accent wordmark used in the banner and section headers.
	Title = lipgloss.NewStyle().Foreground(accentColor).Bold(true)

	// Key styles a placeholder/variable name; Count styles a match count.
	Key   = lipgloss.NewStyle().Bold(true)
	Count = lipgloss.NewStyle().Foreground(mutedColor)

	// panelBorder frames boxed sections (banner, summaries) with a hairline in
	// the muted line color, echoing the web kit's 1px borders.
	panelBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor).
			Padding(0, 1)
)

// Status glyphs. Plain-ASCII friendly, colored via the styles above.
const (
	GlyphOK    = "✓"
	GlyphErr   = "✗"
	GlyphArrow = "→"
	GlyphDot   = "•"
)

// Colors exposes the raw brand colors for consumers that build their own
// lipgloss styles (the huh theme and the TUI).
func Colors() (accent, muted, success, errC, warn, info lipgloss.TerminalColor) {
	return accentColor, mutedColor, successColor, errorColor, warnColor, infoColor
}
