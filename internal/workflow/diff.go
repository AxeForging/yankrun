package workflow

import (
	"os"
	"path/filepath"

	"github.com/AxeForging/yankrun/domain"
	"github.com/pmezard/go-difflib/difflib"
)

// attachDiffs fills FileSummary.Diff for every scanned file whose content would
// change under final, producing a standard unified diff. It is used on dry runs
// so both humans and agents can preview the exact edits without writing files.
func (e Engine) attachDiffs(dir string, settings TemplateSettings, summary *Summary, final domain.InputReplacement) {
	if e.Replacer == nil || len(final.Variables) == 0 {
		return
	}
	for i := range summary.Files {
		rel := summary.Files[i].Path
		data, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			continue
		}
		before := string(data)
		after, n := e.Replacer.ReplaceContent(before, final, settings.StartDelim, settings.EndDelim)
		if n == 0 || after == before {
			continue
		}
		summary.Files[i].Diff = unifiedDiff(rel, before, after)
	}
}

// unifiedDiff renders a git-style unified diff between before and after with a
// small amount of surrounding context.
func unifiedDiff(path, before, after string) string {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(before),
		B:        difflib.SplitLines(after),
		FromFile: "a/" + path,
		ToFile:   "b/" + path,
		Context:  2,
	}
	text, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return ""
	}
	return text
}
