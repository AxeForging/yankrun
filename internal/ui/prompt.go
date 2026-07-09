package ui

import (
	"errors"
	"fmt"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/helpers"
	"github.com/charmbracelet/huh"
)

// Theme returns the AxeForge huh theme: the forge accent on titles, selectors,
// and selected options, muted descriptions, and error red. Callers pass it to
// every huh.Form so interactive prompts match the rest of the CLI.
func Theme() *huh.Theme {
	t := huh.ThemeBase()
	accent, muted, _, errC, _, _ := Colors()

	t.Focused.Title = t.Focused.Title.Foreground(accent).Bold(true)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(accent).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(muted)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(accent)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(accent)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Background(accent)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(errC)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(errC)

	t.Blurred.Title = t.Blurred.Title.Foreground(muted)
	t.Blurred.Description = t.Blurred.Description.Foreground(muted)
	return t
}

// promptErr normalizes a huh error: a user abort (Ctrl+C / Esc) becomes a
// cancelled exit code, anything else is returned as-is.
func promptErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, huh.ErrUserAborted) {
		return helpers.CancelledErr("cancelled")
	}
	return err
}

// PromptValues asks for a value for each key using a single form. Manifest
// metadata drives each field: a description as help text, an enum as a select,
// and required as validation. base seeds each field's initial value.
func PromptValues(manifest *domain.Manifest, keys []string, base map[string]string) (map[string]string, error) {
	ptrs := make(map[string]*string, len(keys))
	fields := make([]huh.Field, 0, len(keys))

	for _, k := range keys {
		init := base[k]
		ptrs[k] = &init
		mv := manifest.Variable(k)

		if mv != nil && len(mv.Enum) > 0 {
			sel := huh.NewSelect[string]().
				Title(k).
				Options(huh.NewOptions(mv.Enum...)...).
				Value(ptrs[k])
			if mv.Description != "" {
				sel = sel.Description(mv.Description)
			}
			fields = append(fields, sel)
			continue
		}

		in := huh.NewInput().Title(k).Value(ptrs[k])
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

	form := huh.NewForm(huh.NewGroup(fields...)).WithTheme(Theme())
	if err := form.Run(); err != nil {
		return nil, promptErr(err)
	}

	values := make(map[string]string, len(keys))
	for k, p := range ptrs {
		if *p != "" {
			values[k] = *p
		}
	}
	return values, nil
}

// SelectOption is one choice in a Select prompt: Label is shown, Value returned.
type SelectOption struct {
	Label string
	Value string
}

// Select asks the user to pick one option.
func Select(title string, options []SelectOption) (string, error) {
	var chosen string
	opts := make([]huh.Option[string], 0, len(options))
	for _, o := range options {
		opts = append(opts, huh.NewOption(o.Label, o.Value))
	}
	field := huh.NewSelect[string]().Title(title).Options(opts...).Value(&chosen)
	if len(opts) > 8 {
		field = field.Filtering(true)
	}
	form := huh.NewForm(huh.NewGroup(field)).WithTheme(Theme())
	if err := form.Run(); err != nil {
		return "", promptErr(err)
	}
	return chosen, nil
}

// Input asks for a single line of text, seeded with def.
func Input(title, description, def string) (string, error) {
	value := def
	field := huh.NewInput().Title(title).Value(&value)
	if description != "" {
		field = field.Description(description)
	}
	form := huh.NewForm(huh.NewGroup(field)).WithTheme(Theme())
	if err := form.Run(); err != nil {
		return "", promptErr(err)
	}
	if value == "" {
		return def, nil
	}
	return value, nil
}

// Confirm asks a yes/no question, defaulting to def.
func Confirm(title string, def bool) (bool, error) {
	value := def
	field := huh.NewConfirm().Title(title).Value(&value)
	form := huh.NewForm(huh.NewGroup(field)).WithTheme(Theme())
	if err := form.Run(); err != nil {
		return false, promptErr(err)
	}
	return value, nil
}
