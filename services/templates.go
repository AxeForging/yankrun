package services

import (
	"context"
	"strings"

	"github.com/AxeForging/yankrun/domain"
)

// ListTemplates returns the configured template repos plus any discovered via
// GitHub. It is the single source of truth shared by the web workbench and the
// MCP server so both surface the same catalog.
func ListTemplates(ctx context.Context, cfg *domain.Config) []domain.TemplateRepo {
	if cfg == nil {
		return nil
	}
	out := make([]domain.TemplateRepo, 0, len(cfg.Templates))
	out = append(out, cfg.Templates...)
	if cfg.GitHub.User != "" || len(cfg.GitHub.Orgs) > 0 {
		if found, err := NewGitHubClient().ListRepos(ctx, cfg.GitHub); err == nil {
			for _, r := range found {
				out = append(out, domain.TemplateRepo{
					Name:          r.FullName,
					URL:           r.SSHURL,
					Description:   r.Description,
					DefaultBranch: r.DefaultBranch,
				})
			}
		}
	}
	return out
}

// FindTemplate resolves a template by exact name/URL or substring match.
func FindTemplate(ctx context.Context, cfg *domain.Config, needle string) (domain.TemplateRepo, bool) {
	needle = strings.ToLower(strings.TrimSpace(needle))
	if needle == "" {
		return domain.TemplateRepo{}, false
	}
	for _, t := range ListTemplates(ctx, cfg) {
		name, url := strings.ToLower(t.Name), strings.ToLower(t.URL)
		if name == needle || url == needle || strings.Contains(name, needle) || strings.Contains(url, needle) {
			return t, true
		}
	}
	return domain.TemplateRepo{}, false
}
