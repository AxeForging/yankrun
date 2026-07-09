package tui

import (
	"fmt"
	"strings"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/internal/ui"
	"github.com/AxeForging/yankrun/internal/workflow"
	"github.com/AxeForging/yankrun/services"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

type state int

const (
	stateScanning state = iota
	stateForm
	statePreviewLoading
	statePreview
	stateApplying
	stateDone
	stateError
	stateEmpty
)

// model drives the interactive templating workbench: scan → fill values →
// preview diffs → apply. It holds no business logic; every data operation goes
// through workflow.Engine.
type model struct {
	engine   workflow.Engine
	dir      string
	settings workflow.TemplateSettings
	provided domain.InputReplacement
	dryRun   bool

	state    state
	spinner  spinner.Model
	form     *huh.Form
	ptrs     map[string]*string
	summary  workflow.Summary
	result   workflow.ApplyResult
	viewport viewport.Model
	err      error

	width, height int
	ready         bool
}

// Messages carrying async work back into the update loop.
type (
	scanDoneMsg    struct{ summary workflow.Summary }
	previewDoneMsg struct{ result workflow.ApplyResult }
	applyDoneMsg   struct{ result workflow.ApplyResult }
	errMsg         struct{ err error }
)

func newModel(engine workflow.Engine, dir string, settings workflow.TemplateSettings, provided domain.InputReplacement, dryRun bool) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = titleStyle
	return model{
		engine:   engine,
		dir:      dir,
		settings: settings,
		provided: provided,
		dryRun:   dryRun,
		state:    stateScanning,
		spinner:  sp,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.scanCmd())
}

// scanCmd runs the read-only scan off the update loop.
func (m model) scanCmd() tea.Cmd {
	return func() tea.Msg {
		summary, err := m.engine.ScanDir(m.dir, m.settings, m.provided)
		if err != nil {
			return errMsg{err}
		}
		return scanDoneMsg{summary}
	}
}

// previewCmd runs a dry-run apply to compute per-file diffs.
func (m model) previewCmd(values map[string]string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.engine.ApplyDir(m.dir, m.settings, domain.InputReplacement{}, values, true, false)
		if err != nil {
			return errMsg{err}
		}
		return previewDoneMsg{result}
	}
}

// applyCmd performs the real apply (honoring the global --dryRun).
func (m model) applyCmd(values map[string]string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.engine.ApplyDir(m.dir, m.settings, domain.InputReplacement{}, values, m.dryRun, false)
		if err != nil {
			return errMsg{err}
		}
		return applyDoneMsg{result}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.viewport = viewport.New(msg.Width-4, maxInt(msg.Height-10, 3))
		m.ready = true
		if m.state == statePreview {
			m.viewport.SetContent(m.diffContent())
		}
		return m, nil

	case tea.KeyMsg:
		// Global quit, except while the form owns the keyboard.
		if m.state != stateForm && key.Matches(msg, keys.Quit) {
			return m, tea.Quit
		}

	case errMsg:
		m.err = msg.err
		m.state = stateError
		return m, nil

	case scanDoneMsg:
		m.summary = msg.summary
		if len(m.summary.Keys) == 0 {
			m.state = stateEmpty
			return m, nil
		}
		m.buildForm()
		m.state = stateForm
		return m, m.form.Init()

	case previewDoneMsg:
		m.result = msg.result
		m.state = statePreview
		if m.ready {
			m.viewport.SetContent(m.diffContent())
		}
		return m, nil

	case applyDoneMsg:
		m.result = msg.result
		m.state = stateDone
		return m, nil
	}

	switch m.state {
	case stateScanning, statePreviewLoading, stateApplying:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case stateForm:
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}
		if m.form.State == huh.StateCompleted {
			m.state = statePreviewLoading
			return m, tea.Batch(m.spinner.Tick, m.previewCmd(m.resolvedValues()))
		}
		if m.form.State == huh.StateAborted {
			return m, tea.Quit
		}
		return m, cmd

	case statePreview:
		if km, ok := msg.(tea.KeyMsg); ok {
			switch {
			case key.Matches(km, keys.Apply):
				m.state = stateApplying
				return m, tea.Batch(m.spinner.Tick, m.applyCmd(m.resolvedValues()))
			case key.Matches(km, keys.Edit):
				m.buildForm()
				m.state = stateForm
				return m, m.form.Init()
			}
		}
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

// buildForm constructs the huh value form seeded with resolved defaults.
func (m *model) buildForm() {
	base := m.resolvedBase()
	m.ptrs = make(map[string]*string, len(m.summary.Keys))
	fields := make([]huh.Field, 0, len(m.summary.Keys))
	for _, k := range m.summary.Keys {
		init := base[k]
		m.ptrs[k] = &init
		mv := m.summary.Manifest.Variable(k)
		if mv != nil && len(mv.Enum) > 0 {
			sel := huh.NewSelect[string]().Title(k).Options(huh.NewOptions(mv.Enum...)...).Value(m.ptrs[k])
			if mv.Description != "" {
				sel = sel.Description(mv.Description)
			}
			fields = append(fields, sel)
			continue
		}
		in := huh.NewInput().Title(k).Value(m.ptrs[k])
		if mv != nil {
			if mv.Description != "" {
				in = in.Description(mv.Description)
			}
			if mv.Required {
				key := k
				in = in.Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("%s is required", key)
					}
					return nil
				})
			}
		}
		fields = append(fields, in)
	}
	m.form = huh.NewForm(huh.NewGroup(fields...)).WithTheme(ui.Theme()).WithShowHelp(true)
}

// resolvedBase is the pre-answer value map (defaults < file < env).
func (m model) resolvedBase() map[string]string {
	return workflow.ResolveValues(m.summary.Manifest, valuesFromInput(m.provided), services.EnvValues(), nil)
}

// resolvedValues folds the current form answers over the base.
func (m model) resolvedValues() map[string]string {
	answers := map[string]string{}
	for k, p := range m.ptrs {
		if *p != "" {
			answers[k] = *p
		}
	}
	return workflow.ResolveValues(m.summary.Manifest, valuesFromInput(m.provided), services.EnvValues(), answers)
}

func valuesFromInput(in domain.InputReplacement) map[string]string {
	values := map[string]string{}
	for _, r := range in.Variables {
		values[r.Key] = r.Value
	}
	return values
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// diffContent renders the per-file diffs into the viewport body.
func (m model) diffContent() string {
	var b strings.Builder
	any := false
	for _, f := range m.result.Summary.Files {
		if f.Diff == "" {
			continue
		}
		any = true
		fmt.Fprintln(&b, titleStyle.Render(f.Path))
		for _, line := range strings.Split(strings.TrimRight(f.Diff, "\n"), "\n") {
			switch {
			case strings.HasPrefix(line, "+"):
				fmt.Fprintln(&b, okStyle.Render(line))
			case strings.HasPrefix(line, "-"):
				fmt.Fprintln(&b, errStyle.Render(line))
			default:
				fmt.Fprintln(&b, subtleStyle.Render(line))
			}
		}
		fmt.Fprintln(&b)
	}
	if !any {
		return subtleStyle.Render("No changes — values leave every file untouched.")
	}
	return b.String()
}
