package tui

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/internal/workflow"
	"github.com/AxeForging/yankrun/services"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func newTestModel(t *testing.T, dir string, dryRun bool) model {
	t.Helper()
	fs := &services.OsFileSystem{}
	engine := workflow.Engine{Replacer: &services.FileReplacer{FileSystem: fs}}
	settings := workflow.TemplateSettings{StartDelim: "[[", EndDelim: "]]", FileSizeLimit: "3 mb"}
	return newModel(engine, dir, settings, domain.InputReplacement{}, dryRun)
}

func waitFor(t *testing.T, tm *teatest.TestModel, want string) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte(want))
	}, teatest.WithDuration(5*time.Second))
}

// TestTUI_FillPreviewApply drives the full workbench flow against a real temp
// directory: scan finds the placeholder, the form fills it, the preview shows
// the diff, and applying writes the resolved value to disk.
func TestTUI_FillPreviewApply(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.txt")
	if err := os.WriteFile(path, []byte("App: [[NAME]]"), 0o600); err != nil {
		t.Fatal(err)
	}

	tm := teatest.NewTestModel(t, newTestModel(t, dir, false), teatest.WithInitialTermSize(100, 30))

	// Form appears with the discovered key.
	waitFor(t, tm, "NAME")
	tm.Type("MyApp")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Preview shows the diff with the new value.
	waitFor(t, tm, "+App: MyApp")

	// Apply, then quit.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	waitFor(t, tm, "Applied")
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "App: MyApp" {
		t.Fatalf("apply wrote %q, want %q", string(got), "App: MyApp")
	}
}

// TestTUI_DryRunDoesNotModify confirms that with --dryRun the apply step never
// touches the file.
func TestTUI_DryRunDoesNotModify(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.txt")
	if err := os.WriteFile(path, []byte("App: [[NAME]]"), 0o600); err != nil {
		t.Fatal(err)
	}

	tm := teatest.NewTestModel(t, newTestModel(t, dir, true), teatest.WithInitialTermSize(100, 30))
	waitFor(t, tm, "NAME")
	tm.Type("MyApp")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitFor(t, tm, "+App: MyApp")
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	waitFor(t, tm, "Dry run")
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))

	if got, _ := os.ReadFile(path); string(got) != "App: [[NAME]]" {
		t.Fatalf("dry run modified file: %q", string(got))
	}
}

// TestTUI_EmptyDirExits shows the empty state and quits cleanly when there are
// no placeholders.
func TestTUI_EmptyDirExits(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "plain.txt"), []byte("nothing here"), 0o600); err != nil {
		t.Fatal(err)
	}
	tm := teatest.NewTestModel(t, newTestModel(t, dir, false), teatest.WithInitialTermSize(100, 30))
	waitFor(t, tm, "No placeholders found")
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
}
