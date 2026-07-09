package helpers

import (
	"os"

	"github.com/mattn/go-isatty"
)

// IsInteractive reports whether yankrun is attached to a real terminal on both
// stdin and stdout. Interactive prompts (huh forms, the TUI, yes/no gates) must
// only run when this is true; otherwise agents and CI would hang on a prompt
// that can never be answered.
func IsInteractive() bool {
	return isTTY(os.Stdin) && isTTY(os.Stdout)
}

// StdoutIsTTY reports whether stdout is a terminal. Used to decide colored vs
// plain output independently of stdin (e.g. `yankrun scan | cat`).
func StdoutIsTTY() bool { return isTTY(os.Stdout) }

// ColorEnabled reports whether styled/colored output should be emitted. It
// honors the NO_COLOR convention (https://no-color.org) and falls back to
// plain output whenever stdout is not a terminal, which keeps piped and
// golden-file output stable.
func ColorEnabled() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	return StdoutIsTTY()
}

func isTTY(f *os.File) bool {
	if f == nil {
		return false
	}
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}
