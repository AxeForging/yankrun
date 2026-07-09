package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// SectionHeader renders an accented, bold section title with a trailing rule,
// e.g. "▸ Discovered placeholders".
func SectionHeader(title string) string {
	return Title.Render(GlyphArrow+" ") + Bold.Render(title)
}

// StatusOK renders a success line: green ✓ + message.
func StatusOK(msg string) string {
	return Success.Render(GlyphOK) + " " + msg
}

// StatusErr renders a failure line: red ✗ + message.
func StatusErr(msg string) string {
	return Error.Render(GlyphErr) + " " + msg
}

// StatusInfo renders a neutral bullet line.
func StatusInfo(msg string) string {
	return Muted.Render(GlyphDot) + " " + msg
}

// Panel frames content in a hairline rounded box with an optional title.
func Panel(title, content string) string {
	if title != "" {
		content = Bold.Render(title) + "\n" + content
	}
	return panelBorder.Render(content)
}

// keyValue right-pads key to width and joins with value using the accent arrow.
func keyValue(key, value string, width int) string {
	padded := Key.Render(key)
	if pad := width - lipgloss.Width(key); pad > 0 {
		padded += strings.Repeat(" ", pad)
	}
	return "  " + padded + "  " + value
}

// writeLines writes each line to w followed by a newline.
func writeLines(w io.Writer, lines ...string) {
	for _, l := range lines {
		fmt.Fprintln(w, l)
	}
}
