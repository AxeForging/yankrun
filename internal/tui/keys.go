package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap holds the bindings used outside of the huh form (which owns its own
// keys while it is focused).
type keyMap struct {
	Apply key.Binding
	Edit  key.Binding
	Quit  key.Binding
}

var keys = keyMap{
	Apply: key.NewBinding(
		key.WithKeys("a", "enter"),
		key.WithHelp("a", "apply"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit values"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c", "esc"),
		key.WithHelp("q", "quit"),
	),
}
