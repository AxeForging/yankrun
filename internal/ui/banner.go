package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
)

// mark is the yankrun terminal mark: an accented double-bit axe glyph echoing
// the brand logo, kept to box-drawing characters so it renders everywhere.
const mark = "⟦y⟧"

// Banner returns the branded header: accent mark + wordmark + tagline. It
// degrades to plain text when color is disabled (lipgloss emits no escapes on a
// non-TTY / NO_COLOR profile).
func Banner() string {
	word := Title.Render("yankrun")
	tag := Muted.Render("template smarter")
	head := lipgloss.JoinHorizontal(lipgloss.Left, Accent.Render(mark), " ", word, "  ", tag)
	return head
}

// PrintBanner writes the banner followed by a blank line to w.
func PrintBanner(w io.Writer) {
	fmt.Fprintln(w, Banner())
	fmt.Fprintln(w)
}

// ServeBanner returns the boxed, on-brand splash printed when the web workbench
// comes up — forge flavor, address front and center.
func ServeBanner(url string) string {
	title := Accent.Render(mark) + "  " + Title.Render("yankrun") + Muted.Render(" — the anvil is hot")
	link := Accent.Render(GlyphArrow) + " " + lipgloss.NewStyle().Bold(true).Underline(true).Render(url)
	line1 := Muted.Render("scan " + GlyphDot + " fill " + GlyphDot + " preview " + GlyphDot + " apply — hammer your templates into shape")
	line2 := Muted.Render("Ctrl+C to bank the coals.")

	body := lipgloss.JoinVertical(lipgloss.Left, title, "", link, "", line1, line2)
	return panelBorder.Padding(1, 2).Render(body)
}

// PrintServeBanner writes the serve splash to w.
func PrintServeBanner(w io.Writer, url string) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, ServeBanner(url))
	fmt.Fprintln(w)
}
