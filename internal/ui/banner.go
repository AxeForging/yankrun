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
