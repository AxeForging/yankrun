package services

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"

    "github.com/AxeForging/yankrun/domain"
)

type RepoInfo struct {
    Name          string
    FullName      string
    HTMLURL       string
    SSHURL        string
    DefaultBranch string
    Description   string
}

// GitHubClient is a minimal client using net/http for listing repos by user/org
type GitHubClient struct { httpClient *http.Client }

func NewGitHubClient() *GitHubClient { return &GitHubClient{httpClient: http.DefaultClient} }

// ListRepos returns template repos discovered from GitHub config (user/orgs), filtered by Topic/Prefix if provided
func (c *GitHubClient) ListRepos(ctx context.Context, gh domain.GitHubConfig) ([]RepoInfo, error) {
    var repos []RepoInfo
    authHeader := ""
    if gh.Token != "" { authHeader = "token " + gh.Token }

    fetch := func(url string) error {
        req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
        if authHeader != "" { req.Header.Set("Authorization", authHeader) }
        req.Header.Set("Accept", "application/vnd.github+json")
        resp, err := c.httpClient.Do(req)
        if err != nil { return err }
        defer resp.Body.Close()
        if resp.StatusCode != 200 { return fmt.Errorf("github api status %d", resp.StatusCode) }
        var arr []struct {
            Name          string   `json:"name"`
            FullName      string   `json:"full_name"`
            HTMLURL       string   `json:"html_url"`
            SSHURL        string   `json:"ssh_url"`
            DefaultBranch string   `json:"default_branch"`
            Description   string   `json:"description"`
            Topics        []string `json:"topics"`
        }
        if err := json.NewDecoder(resp.Body).Decode(&arr); err != nil { return err }
        for _, r := range arr {
            if gh.Topic != "" {
                ok := false
                for _, t := range r.Topics { if t == gh.Topic { ok = true; break } }
                if !ok { continue }
            }
            if gh.Prefix != "" && !strings.HasPrefix(strings.ToLower(r.Name), strings.ToLower(gh.Prefix)) { continue }
            repos = append(repos, RepoInfo{
                Name: r.Name, FullName: r.FullName, HTMLURL: r.HTMLURL, SSHURL: r.SSHURL,
                DefaultBranch: r.DefaultBranch, Description: r.Description,
            })
        }
        return nil
    }

    var lastErr error
    if gh.User != "" {
        vis := "public"
        if gh.IncludePrivate { vis = "all" }
        if err := fetch(fmt.Sprintf("https://api.github.com/users/%s/repos?per_page=100&visibility=%s", gh.User, vis)); err != nil {
            lastErr = fmt.Errorf("failed to list repos for user %s: %w", gh.User, err)
        }
    }
    for _, org := range gh.Orgs {
        if err := fetch(fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=100&type=all", org)); err != nil {
            lastErr = fmt.Errorf("failed to list repos for org %s: %w", org, err)
        }
    }
    if len(repos) == 0 && lastErr != nil {
        return repos, lastErr
    }
    return repos, nil
}


